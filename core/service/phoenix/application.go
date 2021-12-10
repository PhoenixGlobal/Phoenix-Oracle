package phoenix

import (
	"bytes"
	"context"
	stderr "errors"
	"fmt"
	"math/big"
	"os"
	"os/signal"
	"reflect"
	"sync"
	"syscall"

	"PhoenixOracle/core/chain/evm"
	"PhoenixOracle/core/keystore"
	"PhoenixOracle/core/log"
	"PhoenixOracle/core/service"
	"PhoenixOracle/core/service/balancemonitor"
	"PhoenixOracle/core/service/ethereum"
	"PhoenixOracle/core/service/feedmanager"
	"PhoenixOracle/core/service/job"
	"PhoenixOracle/core/service/jobs/fluxmonitor"
	"PhoenixOracle/core/service/jobs/offchainreporting"
	"PhoenixOracle/core/service/jobs/request"
	"PhoenixOracle/core/service/jobs/timer"
	"PhoenixOracle/core/service/jobs/webhook"
	"PhoenixOracle/core/service/pipeline"
	"PhoenixOracle/core/service/txmanager"
	"PhoenixOracle/core/service/vrf"
	strpkg "PhoenixOracle/db"
	"PhoenixOracle/db/config"
	"PhoenixOracle/lib/gracefulpanic"
	"PhoenixOracle/lib/headtracker"
	httypes "PhoenixOracle/lib/headtracker/types"
	"PhoenixOracle/lib/health"
	loggerPkg "PhoenixOracle/lib/logger"
	"PhoenixOracle/lib/nodeversion"
	"PhoenixOracle/lib/postgres"
	"PhoenixOracle/lib/telemetry"
	"PhoenixOracle/lib/telemetry/synchronization"
	"PhoenixOracle/lib/timerbackup"
	"PhoenixOracle/util"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/gobuffalo/packr"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
	"go.uber.org/multierr"
	"go.uber.org/zap/zapcore"
	"gopkg.in/guregu/null.v4"
	"gorm.io/gorm"
)

type Application interface {
	Start() error
	Stop() error
	GetLogger() *loggerPkg.Logger
	GetHealthChecker() health.Checker
	GetStore() *strpkg.Store
	GetEthClient() ethereum.Client
	GetConfig() config.GeneralConfig

	GetEVMConfig() config.EVMConfig
	GetKeyStore() keystore.Master
	GetHeadBroadcaster() httypes.HeadBroadcasterRegistry
	WakeSessionReaper()
	NewBox() packr.Box

	GetExternalInitiatorManager() webhook.ExternalInitiatorManager

	JobSpawner() job.Spawner
	JobORM() job.ORM
	EVMORM() evm.ORM
	PipelineORM() pipeline.ORM
	AddJobV2(ctx context.Context, job job.Job, name null.String) (job.Job, error)
	DeleteJob(ctx context.Context, jobID int32) error
	RunWebhookJobV2(ctx context.Context, jobUUID uuid.UUID, requestBody string, meta pipeline.JSONSerializable) (int64, error)
	ResumeJobV2(ctx context.Context, taskID uuid.UUID, result interface{}) error

	RunJobV2(ctx context.Context, jobID int32, meta map[string]interface{}) (int64, error)
	SetServiceLogger(ctx context.Context, service string, level zapcore.Level) error

	GetFeedsService() feedmanager.Service

	ReplayFromBlock(number uint64) error
}

type PhoenixApplication struct {
	Exiter                   func(int)
	HeadTracker              httypes.Tracker
	HeadBroadcaster          httypes.HeadBroadcaster
	TxManager                txmanager.TxManager
	LogBroadcaster           log.Broadcaster
	EventBroadcaster         postgres.EventBroadcaster
	jobORM                   job.ORM
	jobSpawner               job.Spawner
	pipelineORM              pipeline.ORM
	pipelineRunner           pipeline.Runner
	FeedsService             feedmanager.Service
	webhookJobRunner         webhook.JobRunner
	ethClient                ethereum.Client
	evmORM                   evm.ORM
	Store                    *strpkg.Store
	Config                   config.GeneralConfig
	EVMConfig                config.EVMConfig
	KeyStore                 keystore.Master
	ExternalInitiatorManager webhook.ExternalInitiatorManager
	SessionReaper            utils.SleeperTask
	shutdownOnce             sync.Once
	shutdownSignal           gracefulpanic.Signal
	balanceMonitor           balancemonitor.BalanceMonitor
	explorerClient           synchronization.ExplorerClient
	subservices              []service.Service
	HealthChecker            health.Checker
	logger                   *loggerPkg.Logger

	started     bool
	startStopMu sync.Mutex
}

func NewApplication(logger *loggerPkg.Logger, cfg config.EVMConfig, ethClient ethereum.Client, advisoryLocker postgres.AdvisoryLocker) (Application, error) {
	shutdownSignal := gracefulpanic.NewSignal()
	store, err := strpkg.NewStore(cfg, advisoryLocker, shutdownSignal)
	if err != nil {
		return nil, err
	}
	sqlxDB := postgres.UnwrapGormDB(store.DB)
	gormTxm := postgres.NewGormTransactionManager(store.DB)

	scryptParams := utils.GetScryptParams(cfg)
	keyStore := keystore.New(store.DB, scryptParams)

	setupConfig(cfg, store.DB, keyStore)

	var subservices []service.Service

	telemetryIngressClient := synchronization.TelemetryIngressClient(&synchronization.NoopTelemetryIngressClient{})
	explorerClient := synchronization.ExplorerClient(&synchronization.NoopExplorerClient{})
	monitoringEndpointGen := telemetry.MonitoringEndpointGenerator(&telemetry.NoopAgent{})

	if cfg.ExplorerURL() != nil {
		explorerClient = synchronization.NewExplorerClient(cfg.ExplorerURL(), cfg.ExplorerAccessKey(), cfg.ExplorerSecret(), cfg.StatsPusherLogging())
		monitoringEndpointGen = telemetry.NewExplorerAgent(explorerClient)
	}

	// Use Explorer over TelemetryIngress if both URLs are set
	if cfg.ExplorerURL() == nil && cfg.TelemetryIngressURL() != nil {
		telemetryIngressClient = synchronization.NewTelemetryIngressClient(cfg.TelemetryIngressURL(), cfg.TelemetryIngressServerPubKey(), keyStore.CSA(), cfg.TelemetryIngressLogging())
		monitoringEndpointGen = telemetry.NewIngressAgentWrapper(telemetryIngressClient)
	}
	subservices = append(subservices, explorerClient, telemetryIngressClient)

	if cfg.DatabaseBackupMode() != config.DatabaseBackupModeNone && cfg.DatabaseBackupFrequency() > 0 {
		logger.Infow("DatabaseBackup: periodic database backups are enabled", "frequency", cfg.DatabaseBackupFrequency())

		databaseBackup := timerbackup.NewDatabaseBackup(cfg, logger)
		subservices = append(subservices, databaseBackup)
	} else {
		logger.Info("DatabaseBackup: periodic database backups are disabled. To enable automatic backups, set DATABASE_BACKUP_MODE=lite or DATABASE_BACKUP_MODE=full")
	}

	globalLogger := cfg.CreateProductionLogger()
	globalLogger.SetDB(store.DB)
	serviceLogLevels, err := globalLogger.GetServiceLogLevels()
	if err != nil {
		logger.Fatalf("error getting log levels: %v", err)
	}
	headTrackerLogger, err := globalLogger.InitServiceLevelLogger(loggerPkg.HeadTracker, serviceLogLevels[loggerPkg.HeadTracker])
	if err != nil {
		logger.Fatal("error starting logger for head tracker", err)
	}

	var headBroadcaster httypes.HeadBroadcaster
	var headTracker httypes.Tracker
	if cfg.EthereumDisabled() {
		headBroadcaster = &headtracker.NullBroadcaster{}
		headTracker = &headtracker.NullTracker{}
	} else {
		headBroadcaster = headtracker.NewHeadBroadcaster(logger)
		orm := headtracker.NewORM(store.DB)
		headTracker = headtracker.NewHeadTracker(headTrackerLogger, ethClient, cfg, orm, headBroadcaster)
	}

	eventBroadcaster := postgres.NewEventBroadcaster(cfg.DatabaseURL(), cfg.DatabaseListenerMinReconnectInterval(), cfg.DatabaseListenerMaxReconnectDuration())
	subservices = append(subservices, eventBroadcaster)

	var txManager txmanager.TxManager
	var logBroadcaster log.Broadcaster
	if cfg.EthereumDisabled() {
		txManager = &txmanager.NullTxManager{ErrMsg: "TxManager is not running because Ethereum is disabled"}
		logBroadcaster = &log.NullBroadcaster{ErrMsg: "LogBroadcaster is not running because Ethereum is disabled"}
	} else {
		highestSeenHead, err2 := headTracker.HighestSeenHeadFromDB()
		if err2 != nil {
			return nil, err2
		}

		logBroadcaster = log.NewBroadcaster(log.NewORM(store.DB), ethClient, cfg, logger, highestSeenHead)
		txManager = txmanager.NewBulletproofTxManager(store.DB, ethClient, cfg, keyStore.Eth(),
			advisoryLocker, eventBroadcaster, logger)
		subservices = append(subservices, logBroadcaster, txManager)
	}

	var balanceMonitor balancemonitor.BalanceMonitor
	if cfg.BalanceMonitorEnabled() {
		balanceMonitor = balancemonitor.NewBalanceMonitor(store.DB, ethClient, keyStore.Eth(), logger)
	} else {
		balanceMonitor = &balancemonitor.NullBalanceMonitor{}
	}
	subservices = append(subservices, balanceMonitor)

	promReporter := service.NewPromReporter(store.MustSQLDB())
	subservices = append(subservices, promReporter)

	var (
		pipelineORM    = pipeline.NewORM(store.DB)
		pipelineRunner = pipeline.NewRunner(pipelineORM, cfg, ethClient, keyStore.Eth(), keyStore.VRF(), txManager)
		jobORM         = job.NewORM(store.ORM.DB, cfg, pipelineORM, eventBroadcaster, advisoryLocker, keyStore)
		evmORM         = evm.NewORM(sqlxDB)
	)

	txManager.RegisterResumeCallback(pipelineRunner.ResumeRun)

	var (
		delegates = map[job.Type]job.Delegate{
			job.DirectRequest: request.NewDelegate(
				logger,
				logBroadcaster,
				pipelineRunner,
				pipelineORM,
				ethClient,
				store.DB,
				cfg,
			),
			job.VRF: vrf.NewDelegate(
				store.DB,
				txManager,
				keyStore,
				pipelineRunner,
				pipelineORM,
				logBroadcaster,
				headBroadcaster,
				ethClient,
				cfg,
			),
		}
	)

	// Flux monitor requires ethereum just to boot, silence errors with a null delegate
	if cfg.EthereumDisabled() {
		delegates[job.FluxMonitor] = &job.NullDelegate{Type: job.FluxMonitor}
	} else if cfg.Dev() || cfg.FeatureFluxMonitorV2() {
		delegates[job.FluxMonitor] = fluxmonitor.NewDelegate(
			txManager,
			keyStore.Eth(),
			jobORM,
			pipelineORM,
			pipelineRunner,
			store.DB,
			ethClient,
			logBroadcaster,
			fluxmonitor.Config{
				DefaultHTTPTimeout:             cfg.DefaultHTTPTimeout().Duration(),
				FlagsContractAddress:           cfg.FlagsContractAddress(),
				MinContractPayment:             cfg.MinimumContractPayment(),
				EvmGasLimit:                    cfg.EvmGasLimitDefault(),
				EvmMaxQueuedTransactions:       cfg.EvmMaxQueuedTransactions(),
				FMDefaultTransactionQueueDepth: cfg.FMDefaultTransactionQueueDepth(),
			},
		)
	}

	if (cfg.Dev() && cfg.P2PListenPort() > 0) || cfg.FeatureOffchainReporting() {
		logger.Debug("Off-chain reporting enabled")
		concretePW := offchainreporting.NewSingletonPeerWrapper(keyStore, cfg, store.DB)
		subservices = append(subservices, concretePW)
		delegates[job.OffchainReporting] = offchainreporting.NewDelegate(
			store.DB,
			txManager,
			jobORM,
			cfg,
			keyStore,
			pipelineRunner,
			ethClient,
			logBroadcaster,
			concretePW,
			monitoringEndpointGen,
			cfg.Chain(),
			headBroadcaster,
		)
	} else {
		logger.Debug("Off-chain reporting disabled")
	}

	externalInitiatorManager := webhook.NewExternalInitiatorManager(store.DB, utils.UnrestrictedClient)

	var webhookJobRunner webhook.JobRunner
	if cfg.Dev() || cfg.FeatureWebhookV2() {
		delegate := webhook.NewDelegate(pipelineRunner, externalInitiatorManager)
		delegates[job.Webhook] = delegate
		webhookJobRunner = delegate.WebhookJobRunner()
	}

	if cfg.Dev() || cfg.FeatureCronV2() {
		delegates[job.Cron] = timer.NewDelegate(pipelineRunner)
	}

	jobSpawner := job.NewSpawner(jobORM, cfg, delegates, gormTxm)
	subservices = append(subservices, jobSpawner, pipelineRunner, headBroadcaster)

	feedsORM := feedmanager.NewORM(store.DB)
	verORM := nodeversion.NewORM(postgres.WrapDbWithSqlx(
		postgres.MustSQLDB(store.DB)),
	)
	feedsService := feedmanager.NewService(feedsORM, verORM, gormTxm, jobSpawner, keyStore.CSA(), keyStore.Eth(), cfg)

	healthChecker := health.NewChecker()

	app := &PhoenixApplication{
		ethClient:                ethClient,
		HeadBroadcaster:          headBroadcaster,
		TxManager:                txManager,
		LogBroadcaster:           logBroadcaster,
		EventBroadcaster:         eventBroadcaster,
		jobORM:                   jobORM,
		jobSpawner:               jobSpawner,
		pipelineRunner:           pipelineRunner,
		pipelineORM:              pipelineORM,
		evmORM:                   evmORM,
		FeedsService:             feedsService,
		Config:                   cfg,
		EVMConfig:                cfg,
		webhookJobRunner:         webhookJobRunner,
		Store:                    store,
		KeyStore:                 keyStore,
		SessionReaper:            service.NewSessionReaper(store.DB, cfg),
		Exiter:                   os.Exit,
		ExternalInitiatorManager: externalInitiatorManager,
		shutdownSignal:           shutdownSignal,
		balanceMonitor:           balanceMonitor,
		explorerClient:           explorerClient,
		HealthChecker:            healthChecker,
		HeadTracker:              headTracker,
		logger:                   globalLogger,

		subservices: subservices,
	}

	headBroadcaster.Subscribe(logBroadcaster)
	headBroadcaster.Subscribe(txManager)
	headBroadcaster.Subscribe(promReporter)
	headBroadcaster.Subscribe(balanceMonitor)

	logBroadcaster.AddDependents(1)

	for _, service := range app.subservices {
		if err = app.HealthChecker.Register(reflect.TypeOf(service).String(), service); err != nil {
			return nil, err
		}
	}

	if err = app.HealthChecker.Register(reflect.TypeOf(headTracker).String(), headTracker); err != nil {
		return nil, err
	}

	return app, nil
}

func (app *PhoenixApplication) SetServiceLogger(ctx context.Context, serviceName string, level zapcore.Level) error {
	newL, err := app.logger.InitServiceLevelLogger(serviceName, level.String())
	if err != nil {
		return err
	}

	switch serviceName {
	case loggerPkg.HeadTracker:
		app.HeadTracker.SetLogger(newL)
	case loggerPkg.FluxMonitor:
	case loggerPkg.Keeper:
	default:
		return fmt.Errorf("no service found with name: %s", serviceName)
	}

	return app.logger.Orm.SetServiceLogLevel(ctx, serviceName, level)
}

func setupConfig(cfg config.GeneralConfig, db *gorm.DB, ks keystore.Master) {
	cfg.SetDB(db)
}

func (app *PhoenixApplication) Start() error {
	app.startStopMu.Lock()
	defer app.startStopMu.Unlock()
	if app.started {
		panic("application is already started")
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		select {
		case <-sigs:
		case <-app.shutdownSignal.Wait():
		}
		app.logger.ErrorIf(app.Stop())
		app.Exiter(0)
	}()

	// EthClient must be dialed first because it is required in subtasks
	if err := app.ethClient.Dial(context.Background()); err != nil {
		return err
	}

	if err := app.Store.Start(); err != nil {
		return err
	}

	if err := app.FeedsService.Start(); err != nil {
		app.logger.Infof("[Feeds Service] %v", err)
	}

	for _, subservice := range app.subservices {
		app.logger.Debugw("Starting service...", "serviceType", reflect.TypeOf(subservice))
		if err := subservice.Start(); err != nil {
			return err
		}
	}

	app.LogBroadcaster.DependentReady()

	if err := app.HeadTracker.Start(); err != nil {
		return err
	}

	if err := app.HealthChecker.Start(); err != nil {
		return err
	}

	app.started = true

	return nil
}

func (app *PhoenixApplication) StopIfStarted() error {
	app.startStopMu.Lock()
	defer app.startStopMu.Unlock()
	if app.started {
		return app.stop()
	}
	return nil
}

func (app *PhoenixApplication) Stop() error {
	app.startStopMu.Lock()
	defer app.startStopMu.Unlock()
	return app.stop()
}

func (app *PhoenixApplication) stop() error {
	if !app.started {
		panic("application is already stopped")
	}
	var merr error
	app.shutdownOnce.Do(func() {
		defer func() {
			if err := app.logger.Sync(); err != nil {
				if stderr.Unwrap(err).Error() != os.ErrInvalid.Error() &&
					stderr.Unwrap(err).Error() != "inappropriate ioctl for device" &&
					stderr.Unwrap(err).Error() != "bad file descriptor" {
					merr = multierr.Append(merr, err)
				}
			}
		}()
		app.logger.Info("Gracefully exiting...")

		// Stop services in the reverse order from which they were started

		app.logger.Debug("Stopping HeadTracker...")
		merr = multierr.Append(merr, app.HeadTracker.Stop())

		for i := len(app.subservices) - 1; i >= 0; i-- {
			service := app.subservices[i]
			app.logger.Debugw("Closing service...", "serviceType", reflect.TypeOf(service))
			merr = multierr.Append(merr, service.Close())
		}

		app.logger.Debug("Stopping SessionReaper...")
		merr = multierr.Append(merr, app.SessionReaper.Stop())
		app.logger.Debug("Closing Store...")
		merr = multierr.Append(merr, app.Store.Close())
		app.logger.Debug("Closing HealthChecker...")
		merr = multierr.Append(merr, app.HealthChecker.Close())
		app.logger.Debug("Closing Feeds Service...")
		merr = multierr.Append(merr, app.FeedsService.Close())

		app.logger.Info("Exited all services")

		app.started = false
	})
	return merr
}

// GetStore returns the pointer to the store for the PhoenixApplication.
func (app *PhoenixApplication) GetStore() *strpkg.Store {
	return app.Store
}

func (app *PhoenixApplication) GetEthClient() ethereum.Client {
	return app.ethClient
}

func (app *PhoenixApplication) GetConfig() config.GeneralConfig {
	return app.Config
}

func (app *PhoenixApplication) GetEVMConfig() config.EVMConfig {
	return app.EVMConfig
}

func (app *PhoenixApplication) GetKeyStore() keystore.Master {
	return app.KeyStore
}

func (app *PhoenixApplication) GetLogger() *loggerPkg.Logger {
	return app.logger
}

func (app *PhoenixApplication) GetHealthChecker() health.Checker {
	return app.HealthChecker
}

func (app *PhoenixApplication) JobSpawner() job.Spawner {
	return app.jobSpawner
}

func (app *PhoenixApplication) JobORM() job.ORM {
	return app.jobORM
}

func (app *PhoenixApplication) EVMORM() evm.ORM {
	return app.evmORM
}

func (app *PhoenixApplication) PipelineORM() pipeline.ORM {
	return app.pipelineORM
}

func (app *PhoenixApplication) GetExternalInitiatorManager() webhook.ExternalInitiatorManager {
	return app.ExternalInitiatorManager
}

func (app *PhoenixApplication) GetHeadBroadcaster() httypes.HeadBroadcasterRegistry {
	return app.HeadBroadcaster
}

func (app *PhoenixApplication) WakeSessionReaper() {
	app.SessionReaper.WakeUp()
}

func (app *PhoenixApplication) AddJobV2(ctx context.Context, j job.Job, name null.String) (job.Job, error) {
	return app.jobSpawner.CreateJob(ctx, j, name)
}

func (app *PhoenixApplication) DeleteJob(ctx context.Context, jobID int32) error {
	return app.jobSpawner.DeleteJob(ctx, jobID)
}

func (app *PhoenixApplication) RunWebhookJobV2(ctx context.Context, jobUUID uuid.UUID, requestBody string, meta pipeline.JSONSerializable) (int64, error) {
	return app.webhookJobRunner.RunJob(ctx, jobUUID, requestBody, meta)
}

func (app *PhoenixApplication) RunJobV2(
	ctx context.Context,
	jobID int32,
	meta map[string]interface{},
) (int64, error) {
	if !app.Store.Config.Dev() {
		return 0, errors.New("manual job runs only supported in dev mode - export Phoenix_DEV=true to use")
	}
	jb, err := app.jobORM.FindJob(ctx, jobID)
	if err != nil {
		return 0, errors.Wrapf(err, "job ID %v", jobID)
	}
	var runID int64

	// Some jobs are special in that they do not have a task graph.
	isBootstrap := jb.Type == job.OffchainReporting && jb.OffchainreportingOracleSpec != nil && jb.OffchainreportingOracleSpec.IsBootstrapPeer
	if jb.Type.RequiresPipelineSpec() || !isBootstrap {
		var vars map[string]interface{}
		var saveTasks bool
		if jb.Type == job.VRF {
			saveTasks = true
			// Create a dummy log to trigger a run
			testLog := types.Log{
				Data: bytes.Join([][]byte{
					jb.VRFSpec.PublicKey.MustHash().Bytes(),  // key hash
					common.BigToHash(big.NewInt(42)).Bytes(), // seed
					utils.NewHash().Bytes(),                  // sender
					utils.NewHash().Bytes(),                  // fee
					utils.NewHash().Bytes()},                 // requestID
					[]byte{}),
				Topics:      []common.Hash{{}, jb.ExternalIDEncodeBytesToTopic()}, // jobID BYTES
				TxHash:      utils.NewHash(),
				BlockNumber: 10,
				BlockHash:   utils.NewHash(),
			}
			vars = map[string]interface{}{
				"jobSpec": map[string]interface{}{
					"databaseID":    jb.ID,
					"externalJobID": jb.ExternalJobID,
					"name":          jb.Name.ValueOrZero(),
					"publicKey":     jb.VRFSpec.PublicKey[:],
				},
				"jobRun": map[string]interface{}{
					"meta":           meta,
					"logBlockHash":   testLog.BlockHash[:],
					"logBlockNumber": testLog.BlockNumber,
					"logTxHash":      testLog.TxHash,
					"logTopics":      testLog.Topics,
					"logData":        testLog.Data,
				},
			}
		} else {
			vars = map[string]interface{}{
				"jobRun": map[string]interface{}{
					"meta": meta,
				},
			}
		}
		runID, _, err = app.pipelineRunner.ExecuteAndInsertFinishedRun(ctx, *jb.PipelineSpec, pipeline.NewVarsFrom(vars), *app.logger, saveTasks)
	} else {
		runID, err = app.pipelineRunner.TestInsertFinishedRun(app.Store.DB.WithContext(ctx), jb.ID, jb.Name.String, jb.Type.String(), jb.PipelineSpecID)
	}
	return runID, err
}

func (app *PhoenixApplication) ResumeJobV2(
	ctx context.Context,
	taskID uuid.UUID,
	result interface{},
) error {
	return app.pipelineRunner.ResumeRun(taskID, result)
}

func (app *PhoenixApplication) GetFeedsService() feedmanager.Service {
	return app.FeedsService
}

// NewBox returns the packr.Box instance that holds the static assets to
// be delivered by the router.
func (app *PhoenixApplication) NewBox() packr.Box {
	return packr.NewBox("../../../statics/dist")
}

func (app *PhoenixApplication) ReplayFromBlock(number uint64) error {
	app.LogBroadcaster.ReplayFromBlock(int64(number))
	return nil
}
