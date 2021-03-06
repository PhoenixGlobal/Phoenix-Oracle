package offchainreporting

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"time"

	"PhoenixOracle/util"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/pkg/errors"
	"gorm.io/gorm"

	"PhoenixOracle/core/chain"
	"PhoenixOracle/core/keystore"
	"PhoenixOracle/core/keystore/keys/ethkey"
	"PhoenixOracle/core/keystore/keys/p2pkey"
	"PhoenixOracle/core/log"
	"PhoenixOracle/core/service/ethereum"
	"PhoenixOracle/core/service/job"
	"PhoenixOracle/core/service/pipeline"
	"PhoenixOracle/core/service/txmanager"
	"PhoenixOracle/internal/gethwrappers/generated/offchain_aggregator_wrapper"
	httypes "PhoenixOracle/lib/headtracker/types"
	"PhoenixOracle/lib/libocr/gethwrappers/offchainaggregator"
	ocr "PhoenixOracle/lib/libocr/offchainreporting"
	ocrtypes "PhoenixOracle/lib/libocr/offchainreporting/types"
	"PhoenixOracle/lib/logger"
	"PhoenixOracle/lib/postgres"
	"PhoenixOracle/lib/telemetry"
)

type DelegateConfig interface {
	Chain() *chain.Chain
	ChainID() *big.Int
	Dev() bool
	EvmGasLimitDefault() uint64
	JobPipelineResultWriteQueueDepth() uint64
	OCRBlockchainTimeout() time.Duration
	OCRContractConfirmations() uint16
	OCRContractPollInterval() time.Duration
	OCRContractSubscribeInterval() time.Duration
	OCRContractTransmitterTransmitTimeout() time.Duration
	OCRDatabaseTimeout() time.Duration
	OCRDefaultTransactionQueueDepth() uint32
	OCRKeyBundleID() (string, error)
	OCRObservationGracePeriod() time.Duration
	OCRObservationTimeout() time.Duration
	OCRTraceLogging() bool
	OCRTransmitterAddress() (ethkey.EIP55Address, error)
	P2PBootstrapPeers() ([]string, error)
	P2PPeerID() p2pkey.PeerID
	P2PV2Bootstrappers() []ocrtypes.BootstrapperLocator
	FlagsContractAddress() string
}

type Delegate struct {
	db                    *gorm.DB
	txm                   txManager
	jobORM                job.ORM
	config                DelegateConfig
	keyStore              keystore.Master
	pipelineRunner        pipeline.Runner
	ethClient             ethereum.Client
	logBroadcaster        log.Broadcaster
	peerWrapper           *SingletonPeerWrapper
	monitoringEndpointGen telemetry.MonitoringEndpointGenerator
	chain                 *chain.Chain
	headBroadcaster       httypes.HeadBroadcaster
}

var _ job.Delegate = (*Delegate)(nil)

const ConfigOverriderPollInterval = 30 * time.Second

func NewDelegate(
	db *gorm.DB,
	txm txManager,
	jobORM job.ORM,
	config DelegateConfig,
	keyStore keystore.Master,
	pipelineRunner pipeline.Runner,
	ethClient ethereum.Client,
	logBroadcaster log.Broadcaster,
	peerWrapper *SingletonPeerWrapper,
	monitoringEndpointGen telemetry.MonitoringEndpointGenerator,
	chain *chain.Chain,
	headBroadcaster httypes.HeadBroadcaster,
) *Delegate {
	return &Delegate{
		db,
		txm,
		jobORM,
		config,
		keyStore,
		pipelineRunner,
		ethClient,
		logBroadcaster,
		peerWrapper,
		monitoringEndpointGen,
		chain,
		headBroadcaster,
	}
}

func (d Delegate) JobType() job.Type {
	return job.OffchainReporting
}

func (Delegate) AfterJobCreated(spec job.Job)  {}
func (Delegate) BeforeJobDeleted(spec job.Job) {}

func (d Delegate) ServicesForSpec(jobSpec job.Job) (services []job.Service, err error) {
	if jobSpec.OffchainreportingOracleSpec == nil {
		return nil, errors.Errorf("offchainreporting.Delegate expects an *job.OffchainreportingOracleSpec to be present, got %v", jobSpec)
	}
	concreteSpec := *job.LoadDynamicConfigVars(d.config, *jobSpec.OffchainreportingOracleSpec)

	contract, err := offchain_aggregator_wrapper.NewOffchainAggregator(concreteSpec.ContractAddress.Address(), d.ethClient)
	if err != nil {
		return nil, errors.Wrap(err, "could not instantiate NewOffchainAggregator")
	}

	contractFilterer, err := offchainaggregator.NewOffchainAggregatorFilterer(concreteSpec.ContractAddress.Address(), d.ethClient)
	if err != nil {
		return nil, errors.Wrap(err, "could not instantiate NewOffchainAggregatorFilterer")
	}

	contractCaller, err := offchainaggregator.NewOffchainAggregatorCaller(concreteSpec.ContractAddress.Address(), d.ethClient)
	if err != nil {
		return nil, errors.Wrap(err, "could not instantiate NewOffchainAggregatorCaller")
	}

	gormdb, errdb := d.db.DB()
	if errdb != nil {
		return nil, errors.Wrap(errdb, "unable to open sql db")
	}
	ocrdb := NewDB(gormdb, concreteSpec.ID)

	tracker := NewOCRContractTracker(
		contract,
		contractFilterer,
		contractCaller,
		d.ethClient,
		d.logBroadcaster,
		jobSpec.ID,
		*logger.Default,
		d.db,
		ocrdb,
		d.chain,
		d.headBroadcaster,
	)
	services = append(services, tracker)

	var peerID p2pkey.PeerID
	if concreteSpec.P2PPeerID != nil {
		peerID = *concreteSpec.P2PPeerID
	} else {
		k, err2 := d.keyStore.P2P().GetOrFirst(d.config.P2PPeerID().Raw())
		if err2 != nil {
			return nil, err2
		}
		peerID = k.PeerID()
	}

	peerWrapper := d.peerWrapper
	if peerWrapper == nil {
		return nil, errors.New("cannot setup OCR job service, libp2p peer was missing")
	} else if !peerWrapper.IsStarted() {
		return nil, errors.New("peerWrapper is not started. OCR jobs require a started and running peer. Did you forget to specify P2P_LISTEN_PORT?")
	} else if peerWrapper.PeerID != peerID {
		return nil, errors.Errorf("given peer with ID '%s' does not match OCR configured peer with ID: %s", peerWrapper.PeerID.String(), peerID.String())
	}
	var bootstrapPeers []string
	if concreteSpec.P2PBootstrapPeers != nil {
		bootstrapPeers = concreteSpec.P2PBootstrapPeers
	} else {
		bootstrapPeers, err = d.config.P2PBootstrapPeers()
		if err != nil {
			return nil, err
		}
	}
	v2BootstrapPeers := d.config.P2PV2Bootstrappers()

	loggerWith := logger.Default.With(
		"contractAddress", concreteSpec.ContractAddress,
		"jobName", jobSpec.Name.ValueOrZero(),
		"jobID", jobSpec.ID,
	)
	ocrLogger := NewLogger(loggerWith, d.config.OCRTraceLogging(), func(msg string) {
		d.jobORM.RecordError(context.Background(), jobSpec.ID, msg)
	})

	lc := NewLocalConfig(d.config, concreteSpec)
	if err = ocr.SanityCheckLocalConfig(lc); err != nil {
		return nil, err
	}
	logger.Info(fmt.Sprintf("OCR job using local config %+v", lc))

	if concreteSpec.IsBootstrapPeer {
		var bootstrapper *ocr.BootstrapNode
		bootstrapper, err = ocr.NewBootstrapNode(ocr.BootstrapNodeArgs{
			BootstrapperFactory:   peerWrapper.Peer,
			V1Bootstrappers:       bootstrapPeers,
			ContractConfigTracker: tracker,
			Database:              ocrdb,
			LocalConfig:           lc,
			Logger:                ocrLogger,
		})
		if err != nil {
			return nil, errors.Wrap(err, "error calling NewBootstrapNode")
		}
		services = append(services, bootstrapper)
	} else {
		if len(bootstrapPeers) < 1 {
			return nil, errors.New("need at least one bootstrap peer")
		}
		var kb string
		if concreteSpec.EncryptedOCRKeyBundleID != nil {
			kb = concreteSpec.EncryptedOCRKeyBundleID.String()
		} else {
			kb, err = d.config.OCRKeyBundleID()
			if err != nil {
				return nil, err
			}
		}
		ocrkey, err := d.keyStore.OCR().Get(kb)
		if err != nil {
			return nil, err
		}
		contractABI, err := abi.JSON(strings.NewReader(offchainaggregator.OffchainAggregatorABI))
		if err != nil {
			return nil, errors.Wrap(err, "could not get contract ABI JSON")
		}

		var ta ethkey.EIP55Address
		if concreteSpec.TransmitterAddress != nil {
			ta = *concreteSpec.TransmitterAddress
		} else {
			ta, err = d.config.OCRTransmitterAddress()
			if err != nil {
				return nil, err
			}
		}

		strategy := txmanager.NewQueueingTxStrategy(jobSpec.ExternalJobID, d.config.OCRDefaultTransactionQueueDepth())

		contractTransmitter := NewOCRContractTransmitter(
			concreteSpec.ContractAddress.Address(),
			contractCaller,
			contractABI,
			NewTransmitter(d.txm, d.db, ta.Address(), d.config.EvmGasLimitDefault(), strategy),
			d.logBroadcaster,
			tracker,
			d.config.ChainID(),
		)

		runResults := make(chan pipeline.Run, d.config.JobPipelineResultWriteQueueDepth())
		jobSpec.PipelineSpec.JobName = jobSpec.Name.ValueOrZero()
		jobSpec.PipelineSpec.JobID = jobSpec.ID

		var configOverrider ocrtypes.ConfigOverrider
		configOverriderService, err := d.maybeCreateConfigOverrider(loggerWith, concreteSpec.ContractAddress)
		if err != nil {
			return nil, errors.Wrap(err, "Failed to create ConfigOverrider")
		}

		// NOTE: conditional assigning to `configOverrider` is necessary due to the unfortunate fact that assigning `nil` to an
		// interface variable causes `x == nil` checks to always return false, so methods on the interface cannot be safely called then.
		//
		if configOverriderService != nil {
			services = append(services, configOverriderService)
			configOverrider = configOverriderService
		}

		oracle, err := ocr.NewOracle(ocr.OracleArgs{
			Database: ocrdb,
			Datasource: &dataSource{
				pipelineRunner: d.pipelineRunner,
				ocrLogger:      *loggerWith,
				jobSpec:        jobSpec,
				spec:           *jobSpec.PipelineSpec,
				runResults:     runResults,
			},
			LocalConfig:                  lc,
			ContractTransmitter:          contractTransmitter,
			ContractConfigTracker:        tracker,
			PrivateKeys:                  ocrkey,
			BinaryNetworkEndpointFactory: peerWrapper.Peer,
			Logger:                       ocrLogger,
			V1Bootstrappers:              bootstrapPeers,
			V2Bootstrappers:              v2BootstrapPeers,
			MonitoringEndpoint:           d.monitoringEndpointGen.GenMonitoringEndpoint(concreteSpec.ContractAddress.Address()),
			ConfigOverrider:              configOverrider,
		})
		if err != nil {
			return nil, errors.Wrap(err, "error calling NewOracle")
		}
		services = append(services, oracle)

		services = append([]job.Service{NewResultRunSaver(
			postgres.UnwrapGormDB(d.db),
			runResults,
			d.pipelineRunner,
			make(chan struct{}),
			*loggerWith,
		)}, services...)
	}

	return services, nil
}

func (d *Delegate) maybeCreateConfigOverrider(logger *logger.Logger, contractAddress ethkey.EIP55Address) (*ConfigOverriderImpl, error) {
	flagsContractAddress := d.config.FlagsContractAddress()
	if flagsContractAddress != "" {
		flags, err := NewFlags(flagsContractAddress, d.ethClient)
		if err != nil {
			return nil, errors.Wrapf(err,
				"OCR: unable to create Flags contract instance, check address: %s or remove FLAGS_CONTRACT_ADDRESS configuration variable",
				flagsContractAddress,
			)
		}

		ticker := utils.NewPausableTicker(ConfigOverriderPollInterval)
		return NewConfigOverriderImpl(logger, contractAddress, flags, &ticker)
	}
	return nil, nil
}
