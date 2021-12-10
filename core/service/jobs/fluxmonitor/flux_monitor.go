package fluxmonitor

import (
	"context"
	"fmt"
	"math/big"
	mrand "math/rand"
	"reflect"
	"time"

	"PhoenixOracle/core/log"
	"PhoenixOracle/core/service/ethereum"
	"PhoenixOracle/core/service/job"
	"PhoenixOracle/core/service/jobs/fluxmonitor/promfm"
	"PhoenixOracle/core/service/pipeline"
	"PhoenixOracle/db/models"
	"PhoenixOracle/internal/gethwrappers/generated/flags_wrapper"
	"PhoenixOracle/internal/gethwrappers/generated/flux_aggregator_wrapper"
	"PhoenixOracle/lib/gracefulpanic"
	"PhoenixOracle/lib/logger"
	"PhoenixOracle/lib/postgres"
	"PhoenixOracle/util"
	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type PollRequest struct {
	Type      PollRequestType
	Timestamp time.Time
}

type PollRequestType int

const (
	PollRequestTypeUnknown PollRequestType = iota
	PollRequestTypeInitial
	PollRequestTypePoll
	PollRequestTypeIdle
	PollRequestTypeRound
	PollRequestTypeHibernation
	PollRequestTypeRetry
	PollRequestTypeAwaken
	PollRequestTypeDrumbeat
)

const DefaultHibernationPollPeriod = 168 * time.Hour

type FluxMonitor struct {
	contractAddress   common.Address
	oracleAddress     common.Address
	jobSpec           job.Job
	spec              pipeline.Spec
	runner            pipeline.Runner
	db                *gorm.DB
	orm               ORM
	jobORM            job.ORM
	pipelineORM       pipeline.ORM
	keyStore          KeyStoreInterface
	pollManager       *PollManager
	paymentChecker    *PaymentChecker
	contractSubmitter ContractSubmitter
	deviationChecker  *DeviationChecker
	submissionChecker *SubmissionChecker
	flags             Flags
	fluxAggregator    flux_aggregator_wrapper.FluxAggregatorInterface
	logBroadcaster    log.Broadcaster

	logger *logger.Logger

	backlog       *utils.BoundedPriorityQueue
	chProcessLogs chan struct{}

	utils.StartStopOnce
	chStop     chan struct{}
	waitOnStop chan struct{}
}

func NewFluxMonitor(
	pipelineRunner pipeline.Runner,
	jobSpec job.Job,
	spec pipeline.Spec,
	db *gorm.DB,
	orm ORM,
	jobORM job.ORM,
	pipelineORM pipeline.ORM,
	keyStore KeyStoreInterface,
	pollManager *PollManager,
	paymentChecker *PaymentChecker,
	contractAddress common.Address,
	contractSubmitter ContractSubmitter,
	deviationChecker *DeviationChecker,
	submissionChecker *SubmissionChecker,
	flags Flags,
	fluxAggregator flux_aggregator_wrapper.FluxAggregatorInterface,
	logBroadcaster log.Broadcaster,
	fmLogger *logger.Logger,
) (*FluxMonitor, error) {
	fm := &FluxMonitor{
		db:                db,
		runner:            pipelineRunner,
		jobSpec:           jobSpec,
		spec:              spec,
		orm:               orm,
		jobORM:            jobORM,
		pipelineORM:       pipelineORM,
		keyStore:          keyStore,
		pollManager:       pollManager,
		paymentChecker:    paymentChecker,
		contractAddress:   contractAddress,
		contractSubmitter: contractSubmitter,
		deviationChecker:  deviationChecker,
		submissionChecker: submissionChecker,
		flags:             flags,
		logBroadcaster:    logBroadcaster,
		fluxAggregator:    fluxAggregator,
		logger:            fmLogger,
		backlog: utils.NewBoundedPriorityQueue(map[uint]uint{
			// We want reconnecting nodes to be able to submit to a round
			// that hasn't hit maxAnswers yet, as well as the newest round.
			PriorityNewRoundLog:      2,
			PriorityAnswerUpdatedLog: 1,
			PriorityFlagChangedLog:   2,
		}),
		StartStopOnce: utils.StartStopOnce{},
		chProcessLogs: make(chan struct{}, 1),
		chStop:        make(chan struct{}),
		waitOnStop:    make(chan struct{}),
	}

	return fm, nil
}

func NewFromJobSpec(
	jobSpec job.Job,
	db *gorm.DB,
	orm ORM,
	jobORM job.ORM,
	pipelineORM pipeline.ORM,
	keyStore KeyStoreInterface,
	ethClient ethereum.Client,
	logBroadcaster log.Broadcaster,
	pipelineRunner pipeline.Runner,
	cfg Config,
) (*FluxMonitor, error) {
	fmSpec := jobSpec.FluxMonitorSpec

	if !validatePollTimer(fmSpec.PollTimerDisabled, cfg.MinimumPollingInterval(), fmSpec.PollTimerPeriod) {
		return nil, fmt.Errorf(
			"pollTimerPeriod (%s), must be equal or greater than %s",
			fmSpec.PollTimerPeriod,
			cfg.MinimumPollingInterval(),
		)
	}

	// Set up the flux aggregator
	fluxAggregator, err := flux_aggregator_wrapper.NewFluxAggregator(
		fmSpec.ContractAddress.Address(),
		ethClient,
	)
	if err != nil {
		return nil, err
	}

	contractSubmitter := NewFluxAggregatorContractSubmitter(
		fluxAggregator,
		orm,
		keyStore,
		cfg.EvmGasLimit,
	)

	flags, err := NewFlags(cfg.FlagsContractAddress, ethClient)
	logger.ErrorIf(
		err,
		fmt.Sprintf(
			"unable to create Flags contract instance, check address: %s",
			cfg.FlagsContractAddress,
		),
	)

	paymentChecker := &PaymentChecker{
		MinContractPayment: cfg.MinContractPayment,
		MinJobPayment:      fmSpec.MinPayment,
	}

	jobSpec.PipelineSpec.JobID = jobSpec.ID
	jobSpec.PipelineSpec.JobName = jobSpec.Name.ValueOrZero()

	min, err := fluxAggregator.MinSubmissionValue(nil)
	if err != nil {
		return nil, err
	}

	max, err := fluxAggregator.MaxSubmissionValue(nil)
	if err != nil {
		return nil, err
	}

	fmLogger := logger.Default.With(
		"jobID", jobSpec.ID,
		"contract", fmSpec.ContractAddress.Hex(),
	)

	pollManager, err := NewPollManager(
		PollManagerConfig{
			PollTickerInterval:      fmSpec.PollTimerPeriod,
			PollTickerDisabled:      fmSpec.PollTimerDisabled,
			IdleTimerPeriod:         fmSpec.IdleTimerPeriod,
			IdleTimerDisabled:       fmSpec.IdleTimerDisabled,
			DrumbeatSchedule:        fmSpec.DrumbeatSchedule,
			DrumbeatEnabled:         fmSpec.DrumbeatEnabled,
			DrumbeatRandomDelay:     fmSpec.DrumbeatRandomDelay,
			HibernationPollPeriod:   DefaultHibernationPollPeriod, // Not currently configurable
			MinRetryBackoffDuration: 1 * time.Minute,
			MaxRetryBackoffDuration: 1 * time.Hour,
		},
		fmLogger,
	)
	if err != nil {
		return nil, err
	}

	return NewFluxMonitor(
		pipelineRunner,
		jobSpec,
		*jobSpec.PipelineSpec,
		db,
		orm,
		jobORM,
		pipelineORM,
		keyStore,
		pollManager,
		paymentChecker,
		fmSpec.ContractAddress.Address(),
		contractSubmitter,
		NewDeviationChecker(
			float64(fmSpec.Threshold),
			float64(fmSpec.AbsoluteThreshold),
		),
		NewSubmissionChecker(min, max),
		flags,
		fluxAggregator,
		logBroadcaster,
		fmLogger,
	)
}

const (
	PriorityFlagChangedLog   uint = 0
	PriorityNewRoundLog      uint = 1
	PriorityAnswerUpdatedLog uint = 2
)

// Start implements the job.Service interface. It begins the CSP consumer in a
// single goroutine to poll the price adapters and listen to NewRound events.
func (fm *FluxMonitor) Start() error {
	return fm.StartOnce("FluxMonitor", func() error {
		fm.logger.Debug("Starting Flux Monitor for job")

		go gracefulpanic.WrapRecover(func() {
			fm.consume()
		})

		return nil
	})
}

func (fm *FluxMonitor) IsHibernating() bool {
	if !fm.flags.ContractExists() {
		return false
	}

	isFlagLowered, err := fm.flags.IsLowered(fm.contractAddress)
	if err != nil {
		fm.logger.Errorf("unable to determine hibernation status: %v", err)

		return false
	}

	return !isFlagLowered
}

// Close implements the job.Service interface. It stops this instance from
// polling, cleaning up resources.
func (fm *FluxMonitor) Close() error {
	return fm.StopOnce("FluxMonitor", func() error {
		fm.pollManager.Stop()
		close(fm.chStop)
		<-fm.waitOnStop

		return nil
	})
}

func (fm *FluxMonitor) JobID() int32 { return fm.spec.JobID }

func (fm *FluxMonitor) HandleLog(broadcast log.Broadcast) {
	log := broadcast.DecodedLog()
	if log == nil || reflect.ValueOf(log).IsNil() {
		fm.logger.Error("HandleLog: ignoring nil value")
		return
	}

	switch log := log.(type) {
	case *flux_aggregator_wrapper.FluxAggregatorNewRound:
		fm.backlog.Add(PriorityNewRoundLog, broadcast)

	case *flux_aggregator_wrapper.FluxAggregatorAnswerUpdated:
		fm.backlog.Add(PriorityAnswerUpdatedLog, broadcast)

	case *flags_wrapper.FlagsFlagRaised:
		if log.Subject == utils.ZeroAddress || log.Subject == fm.contractAddress {
			fm.backlog.Add(PriorityFlagChangedLog, broadcast)
		}

	case *flags_wrapper.FlagsFlagLowered:
		if log.Subject == utils.ZeroAddress || log.Subject == fm.contractAddress {
			fm.backlog.Add(PriorityFlagChangedLog, broadcast)
		}

	default:
		fm.logger.Warnf("unexpected log type %T", log)
		return
	}

	select {
	case fm.chProcessLogs <- struct{}{}:
	default:
	}
}

func (fm *FluxMonitor) consume() {
	defer close(fm.waitOnStop)

	if err := fm.SetOracleAddress(); err != nil {
		fm.logger.Warnw(
			"unable to set oracle address, this flux monitor job may not work correctly",
			"err", err,
		)
	}

	// Subscribe to contract logs
	unsubscribe := fm.logBroadcaster.Register(fm, log.ListenerOpts{
		Contract: fm.fluxAggregator.Address(),
		ParseLog: fm.fluxAggregator.ParseLog,
		LogsWithTopics: map[common.Hash][][]log.Topic{
			flux_aggregator_wrapper.FluxAggregatorNewRound{}.Topic():      nil,
			flux_aggregator_wrapper.FluxAggregatorAnswerUpdated{}.Topic(): nil,
		},
		NumConfirmations: 0,
	})
	defer unsubscribe()

	if fm.flags.ContractExists() {
		unsubscribe := fm.logBroadcaster.Register(fm, log.ListenerOpts{
			Contract: fm.flags.Address(),
			ParseLog: fm.flags.ParseLog,
			LogsWithTopics: map[common.Hash][][]log.Topic{
				flags_wrapper.FlagsFlagLowered{}.Topic(): nil,
				flags_wrapper.FlagsFlagRaised{}.Topic():  nil,
			},
			NumConfirmations: 0,
		})
		defer unsubscribe()
	}

	fm.pollManager.Start(fm.IsHibernating(), fm.initialRoundState())

	tickLogger := fm.logger.With(
		"pollInterval", fm.pollManager.cfg.PollTickerInterval,
		"idlePeriod", fm.pollManager.cfg.IdleTimerPeriod,
	)

	for {
		select {
		case <-fm.chStop:
			return

		case <-fm.chProcessLogs:
			fm.processLogs()

		case at := <-fm.pollManager.PollTickerTicks():
			tickLogger.Debugf("Poll ticker fired on %v", formatTime(at))
			fm.pollIfEligible(PollRequestTypePoll, fm.deviationChecker, nil)

		case at := <-fm.pollManager.IdleTimerTicks():
			tickLogger.Debugf("Idle timer fired on %v", formatTime(at))
			fm.pollIfEligible(PollRequestTypeIdle, NewZeroDeviationChecker(), nil)

		case at := <-fm.pollManager.RoundTimerTicks():
			tickLogger.Debugf("Round timer fired on %v", formatTime(at))
			fm.pollIfEligible(PollRequestTypeRound, fm.deviationChecker, nil)

		case at := <-fm.pollManager.HibernationTimerTicks():
			tickLogger.Debugf("Hibernation timer fired on %v", formatTime(at))
			fm.pollIfEligible(PollRequestTypeHibernation, NewZeroDeviationChecker(), nil)

		case at := <-fm.pollManager.RetryTickerTicks():
			tickLogger.Debugf("Retry ticker fired on %v", formatTime(at))
			fm.pollIfEligible(PollRequestTypeRetry, NewZeroDeviationChecker(), nil)

		case at := <-fm.pollManager.DrumbeatTicks():
			tickLogger.Debugf("Drumbeat ticker fired on %v", formatTime(at))
			fm.pollIfEligible(PollRequestTypeDrumbeat, NewZeroDeviationChecker(), nil)

		case request := <-fm.pollManager.Poll():
			switch request.Type {
			case PollRequestTypeUnknown:
				break
			default:
				fm.pollIfEligible(request.Type, fm.deviationChecker, nil)
			}
		}
	}
}

func formatTime(at time.Time) string {
	ago := time.Since(at)
	return fmt.Sprintf("%v (%v ago)", at.UTC().Format(time.RFC3339), ago)
}

func (fm *FluxMonitor) SetOracleAddress() error {
	oracleAddrs, err := fm.fluxAggregator.GetOracles(nil)
	if err != nil {
		fm.logger.Error("failed to get list of oracles from FluxAggregator contract")
		return errors.Wrap(err, "failed to get list of oracles from FluxAggregator contract")
	}
	keys, err := fm.keyStore.SendingKeys()
	if err != nil {
		return errors.Wrap(err, "failed to load keys")
	}
	for _, k := range keys {
		for _, oracleAddr := range oracleAddrs {
			if k.Address.Address() == oracleAddr {
				fm.oracleAddress = oracleAddr
				return nil
			}
		}
	}

	log := fm.logger.With(
		"keys", keys,
		"oracleAddresses", oracleAddrs,
	)

	if len(keys) > 0 {
		addr := keys[0].Address.Address()
		log.Warnw("None of the node's keys matched any oracle addresses, using first available key. This flux monitor job may not work correctly",
			"address", addr.Hex(),
		)
		fm.oracleAddress = addr

		return nil
	}

	log.Error("No keys found. This flux monitor job may not work correctly")
	return errors.New("No keys found")
}

func (fm *FluxMonitor) processLogs() {
	for !fm.backlog.Empty() {
		maybeBroadcast := fm.backlog.Take()
		broadcast, ok := maybeBroadcast.(log.Broadcast)
		if !ok {
			fm.logger.Errorf("Failed to convert backlog into LogBroadcast.  Type is %T", maybeBroadcast)
		}
		fm.processBroadcast(broadcast)
	}
}

func (fm *FluxMonitor) processBroadcast(broadcast log.Broadcast) {

	// If the log is a duplicate of one we've seen before, ignore it (this
	// happens because of the LogBroadcaster's backfilling behavior).
	ctx, cancel := postgres.DefaultQueryCtx()
	defer cancel()
	consumed, err := fm.logBroadcaster.WasAlreadyConsumed(fm.db.WithContext(ctx), broadcast)

	if err != nil {
		fm.logger.Errorf("Error determining if log was already consumed: %v", err)
		return
	} else if consumed {
		fm.logger.Debug("Log was already consumed by Flux Monitor, skipping")
		return
	}

	started := time.Now()
	decodedLog := broadcast.DecodedLog()
	switch log := decodedLog.(type) {
	case *flux_aggregator_wrapper.FluxAggregatorNewRound:
		fm.respondToNewRoundLog(*log, broadcast)
	case *flux_aggregator_wrapper.FluxAggregatorAnswerUpdated:
		fm.respondToAnswerUpdatedLog(*log)
		fm.markLogAsConsumed(broadcast, decodedLog, started)
	case *flags_wrapper.FlagsFlagRaised:
		fm.respondToFlagsRaisedLog()
		fm.markLogAsConsumed(broadcast, decodedLog, started)
	case *flags_wrapper.FlagsFlagLowered:
		// Only reactivate if it is hibernating
		if fm.pollManager.cfg.IsHibernating {
			fm.pollManager.Awaken(fm.initialRoundState())
			fm.pollIfEligible(PollRequestTypeAwaken, NewZeroDeviationChecker(), broadcast)
		}
	default:
		fm.logger.Errorf("unknown log %v of type %T", log, log)
	}
}

func (fm *FluxMonitor) markLogAsConsumed(broadcast log.Broadcast, decodedLog interface{}, started time.Time) {
	ctx, cancel := postgres.DefaultQueryCtx()
	defer cancel()
	if err := fm.logBroadcaster.MarkConsumed(fm.db.WithContext(ctx), broadcast); err != nil {
		fm.logger.Errorw("FluxMonitor: failed to mark log as consumed",
			"err", err, "logType", fmt.Sprintf("%T", decodedLog), "log", broadcast.String(), "elapsed", time.Since(started))
	}
}

func (fm *FluxMonitor) respondToFlagsRaisedLog() {
	fm.logger.Debug("FlagsFlagRaised log")
	// check the contract before hibernating, because one flag could be lowered
	// while the other flag remains raised
	isFlagLowered, err := fm.flags.IsLowered(fm.contractAddress)
	fm.logger.ErrorIf(err, "Error determining if flag is still raised")
	if !isFlagLowered {
		fm.pollManager.Hibernate()
	}
}

func (fm *FluxMonitor) respondToAnswerUpdatedLog(log flux_aggregator_wrapper.FluxAggregatorAnswerUpdated) {
	answerUpdatedLogger := fm.logger.With(
		"round", log.RoundId,
		"answer", log.Current.String(),
		"timestamp", log.UpdatedAt.String(),
	)

	answerUpdatedLogger.Debug("AnswerUpdated log")

	roundState, err := fm.roundState(0)
	if err != nil {
		answerUpdatedLogger.Errorf("could not fetch oracleRoundState: %v", err)

		return
	}

	fm.pollManager.Reset(roundState)
}

func (fm *FluxMonitor) respondToNewRoundLog(log flux_aggregator_wrapper.FluxAggregatorNewRound, lb log.Broadcast) {
	started := time.Now()

	newRoundLogger := fm.logger.With(
		"round", log.RoundId,
		"startedBy", log.StartedBy.Hex(),
		"startedAt", log.StartedAt.String(),
		"startedAtUtc", time.Unix(log.StartedAt.Int64(), 0).UTC().Format(time.RFC3339),
	)
	var markConsumed = true
	defer func() {
		if markConsumed {
			if err := fm.logBroadcaster.MarkConsumed(fm.db, lb); err != nil {
				fm.logger.Errorw("FluxMonitor: failed to mark log consumed", "err", err, "log", lb.String())
			}
		}
	}()

	newRoundLogger.Debug("NewRound log")
	promfm.SetBigInt(promfm.SeenRound.WithLabelValues(fmt.Sprintf("%d", fm.spec.JobID)), log.RoundId)

	logRoundID := uint32(log.RoundId.Uint64())

	// We always want to reset the idle timer upon receiving a NewRound log, so we do it before any `return` statements.
	fm.pollManager.ResetIdleTimer(log.StartedAt.Uint64())

	mostRecentRoundID, err := fm.orm.MostRecentFluxMonitorRoundID(fm.contractAddress)
	if err != nil && err != gorm.ErrRecordNotFound {
		newRoundLogger.Errorf("error fetching Flux Monitor most recent round ID from DB: %v", err)
		return
	}

	roundStats, jobRunStatus, err := fm.statsAndStatusForRound(logRoundID, 1)
	if err != nil {
		newRoundLogger.Errorf("error determining round stats / run status for round: %v", err)
		return
	}

	if logRoundID < mostRecentRoundID && roundStats.NumNewRoundLogs > 0 {
		newRoundLogger.Debugf("Received an older round log (and number of previously received NewRound logs is: %v) - "+
			"a possible reorg, hence deleting round ids from %v to %v", roundStats.NumNewRoundLogs, logRoundID, mostRecentRoundID)
		err = fm.orm.DeleteFluxMonitorRoundsBackThrough(fm.contractAddress, logRoundID)
		if err != nil {
			newRoundLogger.Errorf("error deleting reorged Flux Monitor rounds from DB: %v", err)
			return
		}

		// as all newer stats were deleted, at this point a new round stats entry will be created
		roundStats, err = fm.orm.FindOrCreateFluxMonitorRoundStats(fm.contractAddress, logRoundID, 1)
		if err != nil {
			newRoundLogger.Errorf("error determining subsequent round stats for round: %v", err)
			return
		}
	}

	if roundStats.NumSubmissions > 0 {
		// This indicates either that:
		//     - We tried to start a round at the same time as another node, and their transaction was mined first, or
		//     - The chain experienced a shallow reorg that unstarted the current round.
		// If our previous attempt is still pending, return early and don't re-submit
		// If our previous attempt is already over (completed or errored), we should retry
		newRoundLogger.Debugf("There are already %v existing submissions to this round, while job run status is: %v", roundStats.NumSubmissions, jobRunStatus)
		if !jobRunStatus.Finished() {
			newRoundLogger.Debug("Ignoring new round request: started round simultaneously with another node")
			return
		}
	}

	// Ignore rounds we started
	if fm.oracleAddress == log.StartedBy {
		newRoundLogger.Info("Ignoring new round request: we started this round")
		return
	}

	// Ignore rounds we're not eligible for, or for which we won't be paid
	roundState, err := fm.roundState(logRoundID)
	if err != nil {
		newRoundLogger.Errorf("Ignoring new round request: error fetching eligibility from contract: %v", err)
		return
	}

	fm.pollManager.Reset(roundState)
	err = fm.checkEligibilityAndAggregatorFunding(roundState)
	if err != nil {
		newRoundLogger.Infof("Ignoring new round request: %v", err)
		return
	}

	newRoundLogger.Info("Responding to new round request")

	// Best effort to attach metadata.
	var metaDataForBridge map[string]interface{}
	lrd, err := fm.fluxAggregator.LatestRoundData(nil)
	if err != nil {
		newRoundLogger.Warnw("Couldn't read latest round data for request meta", "err", err)
	} else {
		metaDataForBridge, err = models.MarshalBridgeMetaData(lrd.Answer, lrd.UpdatedAt)
		if err != nil {
			newRoundLogger.Warnw("Error marshalling roundState for request meta", "err", err)
		}
	}

	vars := pipeline.NewVarsFrom(map[string]interface{}{
		"jobSpec": map[string]interface{}{
			"databaseID":    fm.jobSpec.ID,
			"externalJobID": fm.jobSpec.ExternalJobID,
			"name":          fm.jobSpec.Name.ValueOrZero(),
		},
		"jobRun": map[string]interface{}{
			"meta": metaDataForBridge,
		},
	})

	// Call the v2 pipeline to execute a new job run
	run, results, err := fm.runner.ExecuteRun(context.Background(), fm.spec, vars, *fm.logger)
	if err != nil {
		logger.Errorw(fmt.Sprintf("error executing new run for job ID %v name %v", fm.spec.JobID, fm.spec.JobName), "err", err)
		return
	}
	result, err := results.FinalResult().SingularResult()
	if err != nil || result.Error != nil {
		logger.Errorw("can't fetch answer", "err", err, "result", result)
		ctx, cancel := postgres.DefaultQueryCtx()
		defer cancel()
		fm.jobORM.RecordError(ctx, fm.spec.JobID, "Error polling")
		return
	}
	answer, err := utils.ToDecimal(result.Value)
	if err != nil {
		logger.Errorw(fmt.Sprintf("error executing new run for job ID %v name %v", fm.spec.JobID, fm.spec.JobName), "err", err)
		return
	}

	if !fm.isValidSubmission(newRoundLogger, answer, started) {
		return
	}

	if roundState.PaymentAmount == nil {
		newRoundLogger.Error("roundState.PaymentAmount shouldn't be nil")
	}

	err = postgres.GormTransactionWithDefaultContext(fm.db, func(tx *gorm.DB) error {
		runID, err2 := fm.runner.InsertFinishedRun(postgres.UnwrapGorm(tx), run, false)
		if err2 != nil {
			return err2
		}
		err2 = fm.queueTransactionForBPTXM(tx, runID, answer, roundState.RoundId, &log)
		if err2 != nil {
			return err2
		}
		return fm.logBroadcaster.MarkConsumed(tx, lb)
	})
	// Either the tx failed and we want to reprocess the log, or it succeeded and already marked it consumed
	markConsumed = false
	if err != nil {
		newRoundLogger.Errorf("unable to create job run: %v", err)
		return
	}
}

var (
	// ErrNotEligible defines when the round is not eligible for submission
	ErrNotEligible = errors.New("not eligible to submit")
	// ErrUnderfunded defines when the aggregator does not have sufficient funds
	ErrUnderfunded = errors.New("aggregator is underfunded")
	// ErrPaymentTooLow defines when the round payment is too low
	ErrPaymentTooLow = errors.New("round payment amount < minimum contract payment")
)

func (fm *FluxMonitor) checkEligibilityAndAggregatorFunding(roundState flux_aggregator_wrapper.OracleRoundState) error {
	if !roundState.EligibleToSubmit {
		return ErrNotEligible
	} else if !fm.paymentChecker.SufficientFunds(
		roundState.AvailableFunds,
		roundState.PaymentAmount,
		roundState.OracleCount,
	) {
		return ErrUnderfunded
	} else if !fm.paymentChecker.SufficientPayment(roundState.PaymentAmount) {
		return ErrPaymentTooLow
	}
	return nil
}

func (fm *FluxMonitor) pollIfEligible(pollReq PollRequestType, deviationChecker *DeviationChecker, broadcast log.Broadcast) {
	started := time.Now()

	l := fm.logger.With(
		"threshold", deviationChecker.Thresholds.Rel,
		"absoluteThreshold", deviationChecker.Thresholds.Abs,
	)
	var markConsumed = true
	defer func() {
		if markConsumed && broadcast != nil {
			if err := fm.logBroadcaster.MarkConsumed(fm.db, broadcast); err != nil {
				l.Errorw("FluxMonitor: failed to mark log consumed", "err", err, "log", broadcast.String())
			}
		}
	}()

	if pollReq != PollRequestTypeHibernation && fm.pollManager.cfg.IsHibernating {
		l.Warnw("FluxMonitor: Skipping poll because a ticker fired while hibernating")
		return
	}

	if !fm.logBroadcaster.IsConnected() {
		l.Warnw("FluxMonitor: LogBroadcaster is not connected to Ethereum node, skipping poll")
		return
	}

	// Ask the FluxAggregator which round we should be submitting to, and what the state of that round is.
	roundState, err := fm.roundState(0)
	if err != nil {
		l.Errorw("unable to determine eligibility to submit from FluxAggregator contract", "err", err)
		fm.jobORM.RecordError(
			context.Background(),
			fm.spec.JobID,
			"Unable to call roundState method on provided contract. Check contract address.",
		)

		return
	}

	l = l.With("reportableRound", roundState.RoundId)

	// Because drumbeat ticker may fire at the same time on multiple nodes, we wait a short random duration
	// after getting a recommended round id, to avoid starting multiple rounds in case of chains with instant tx confirmation
	if pollReq == PollRequestTypeDrumbeat && fm.pollManager.cfg.DrumbeatEnabled && fm.pollManager.cfg.DrumbeatRandomDelay > 0 {
		delay := time.Duration(mrand.Int63n(int64(fm.pollManager.cfg.DrumbeatRandomDelay)))
		l.Infof("waiting %v (of max: %v) before continuing...", delay, fm.pollManager.cfg.DrumbeatRandomDelay)
		time.Sleep(delay)

		roundStateNew, err2 := fm.roundState(roundState.RoundId)
		if err2 != nil {
			l.Errorw("unable to determine eligibility to submit from FluxAggregator contract", "err", err2)
			fm.jobORM.RecordError(
				context.Background(),
				fm.spec.JobID,
				"Unable to call roundState method on provided contract. Check contract address.",
			)

			return
		}
		roundState = roundStateNew
	}

	fm.pollManager.Reset(roundState)
	// Retry if a idle timer fails
	defer func() {
		if pollReq == PollRequestTypeIdle {
			if err != nil {
				if fm.pollManager.StartRetryTicker() {
					min, max := fm.pollManager.retryTicker.Bounds()
					l.Debugw(fmt.Sprintf("started retry ticker (frequency between: %v - %v) because of error: '%v'", min, max, err.Error()))
				}
				return
			}
			fm.pollManager.StopRetryTicker()
		}
	}()

	roundStats, jobRunStatus, err := fm.statsAndStatusForRound(roundState.RoundId, 0)
	if err != nil {
		l.Errorw("error determining round stats / run status for round", "err", err)

		return
	}

	// If we've already successfully submitted to this round (ie through a NewRound log)
	// and the associated JobRun hasn't errored, skip polling
	if roundStats.NumSubmissions > 0 && !jobRunStatus.Errored() {
		l.Infow("skipping poll: round already answered, tx unconfirmed", "jobRunStatus", jobRunStatus)

		return
	}

	// Don't submit if we're not eligible, or won't get paid
	err = fm.checkEligibilityAndAggregatorFunding(roundState)
	if err != nil {
		l.Infof("skipping poll: %v", err)

		return
	}

	var metaDataForBridge map[string]interface{}
	lrd, err := fm.fluxAggregator.LatestRoundData(nil)
	if err != nil {
		l.Warnw("Couldn't read latest round data for request meta", "err", err)
	} else {
		metaDataForBridge, err = models.MarshalBridgeMetaData(lrd.Answer, lrd.UpdatedAt)
		if err != nil {
			l.Warnw("Error marshalling roundState for request meta", "err", err)
		}
	}

	vars := pipeline.NewVarsFrom(map[string]interface{}{
		"jobSpec": map[string]interface{}{
			"databaseID":    fm.jobSpec.ID,
			"externalJobID": fm.jobSpec.ExternalJobID,
			"name":          fm.jobSpec.Name.ValueOrZero(),
		},
		"jobRun": map[string]interface{}{
			"meta": metaDataForBridge,
		},
	})

	run, results, err := fm.runner.ExecuteRun(context.Background(), fm.spec, vars, *fm.logger)
	if err != nil {
		ctx, cancel := postgres.DefaultQueryCtx()
		defer cancel()
		l.Errorw("can't fetch answer", "err", err)
		fm.jobORM.RecordError(ctx, fm.spec.JobID, "Error polling")
		return
	}
	result, err := results.FinalResult().SingularResult()
	if err != nil || result.Error != nil {
		ctx, cancel := postgres.DefaultQueryCtx()
		defer cancel()
		l.Errorw("can't fetch answer", "err", err, "result", result)
		fm.jobORM.RecordError(ctx, fm.spec.JobID, "Error polling")
		return
	}
	answer, err := utils.ToDecimal(result.Value)
	if err != nil {
		logger.Errorw(fmt.Sprintf("error executing new run for job ID %v name %v", fm.spec.JobID, fm.spec.JobName), "err", err)
		return
	}

	if !fm.isValidSubmission(l, answer, started) {
		return
	}

	jobID := fmt.Sprintf("%d", fm.spec.JobID)
	latestAnswer := decimal.NewFromBigInt(roundState.LatestSubmission, 0)
	promfm.SetDecimal(promfm.SeenValue.WithLabelValues(jobID), answer)

	l = l.With(
		"latestAnswer", latestAnswer,
		"answer", answer,
	)

	if roundState.RoundId > 1 && !deviationChecker.OutsideDeviation(latestAnswer, answer) {
		l.Debugw("deviation < threshold, not submitting")
		return
	}

	if roundState.RoundId > 1 {
		l.Infow("deviation > threshold, submitting")
	} else {
		l.Infow("starting first round")
	}

	if roundState.PaymentAmount == nil {
		l.Error("roundState.PaymentAmount shouldn't be nil")
	}

	err = postgres.GormTransactionWithDefaultContext(fm.db, func(tx *gorm.DB) error {
		runID, err2 := fm.runner.InsertFinishedRun(postgres.UnwrapGorm(tx), run, true)
		if err2 != nil {
			return err2
		}
		err2 = fm.queueTransactionForBPTXM(tx, runID, answer, roundState.RoundId, nil)
		if err2 != nil {
			return err2
		}
		if broadcast != nil {
			// In the case of a flag lowered, the pollEligible call is triggered by a log.
			return fm.logBroadcaster.MarkConsumed(tx, broadcast)
		}
		return nil
	})
	// Either the tx failed and we want to reprocess the log, or it succeeded and already marked it consumed
	markConsumed = false
	if err != nil {
		l.Errorw("can't create job run", "err", err)
		return
	}

	promfm.SetDecimal(promfm.ReportedValue.WithLabelValues(jobID), answer)
	promfm.SetUint32(promfm.ReportedRound.WithLabelValues(jobID), roundState.RoundId)
}

func (fm *FluxMonitor) isValidSubmission(l *logger.Logger, answer decimal.Decimal, started time.Time) bool {
	if fm.submissionChecker.IsValid(answer) {
		return true
	}

	l.Errorw("answer is outside acceptable range",
		"min", fm.submissionChecker.Min,
		"max", fm.submissionChecker.Max,
		"answer", answer,
	)
	fm.jobORM.RecordError(context.Background(), fm.spec.JobID, "Answer is outside acceptable range")

	jobId := fm.spec.JobID
	jobName := fm.spec.JobName
	elapsed := time.Since(started)
	pipeline.PromPipelineTaskExecutionTime.WithLabelValues(fmt.Sprintf("%d", jobId), jobName, "", job.FluxMonitor.String()).Set(float64(elapsed))
	pipeline.PromPipelineRunErrors.WithLabelValues(fmt.Sprintf("%d", jobId), jobName).Inc()
	pipeline.PromPipelineRunTotalTimeToCompletion.WithLabelValues(fmt.Sprintf("%d", jobId), jobName).Set(float64(elapsed))
	pipeline.PromPipelineTasksTotalFinished.WithLabelValues(fmt.Sprintf("%d", jobId), jobName, "", job.FluxMonitor.String(), "error").Inc()
	return false
}

func (fm *FluxMonitor) roundState(roundID uint32) (flux_aggregator_wrapper.OracleRoundState, error) {
	return fm.fluxAggregator.OracleRoundState(nil, fm.oracleAddress, roundID)
}

func (fm *FluxMonitor) initialRoundState() flux_aggregator_wrapper.OracleRoundState {
	defaultRoundState := flux_aggregator_wrapper.OracleRoundState{
		StartedAt: uint64(time.Now().Unix()),
	}
	latestRoundData, err := fm.fluxAggregator.LatestRoundData(nil)
	if err != nil {
		fm.logger.Warnf(
			"unable to retrieve latestRoundData for FluxAggregator contract - defaulting "+
				"to current time for tickers: %v",
			err,
		)
		return defaultRoundState
	}
	roundID := uint32(latestRoundData.RoundId.Uint64())
	latestRoundState, err := fm.fluxAggregator.OracleRoundState(nil, fm.oracleAddress, roundID)
	if err != nil {
		fm.logger.Warnf(
			"unable to call roundState for latest round, round: %d, err: %v",
			latestRoundData.RoundId,
			err,
		)
		return defaultRoundState
	}
	return latestRoundState
}

func (fm *FluxMonitor) queueTransactionForBPTXM(db *gorm.DB, runID int64, answer decimal.Decimal, roundID uint32, log *flux_aggregator_wrapper.FluxAggregatorNewRound) error {
	// Submit the Eth Tx
	err := fm.contractSubmitter.Submit(
		db,
		new(big.Int).SetInt64(int64(roundID)),
		answer.BigInt(),
	)
	if err != nil {
		return err
	}

	numLogs := uint(0)
	if log != nil {
		numLogs = 1
	}
	// Update the flux monitor round stats
	err = fm.orm.UpdateFluxMonitorRoundStats(
		db,
		fm.contractAddress,
		roundID,
		runID,
		numLogs,
	)
	if err != nil {
		fm.logger.Errorw(
			fmt.Sprintf("error updating FM round submission count: %v", err),
			"roundID", roundID,
		)

		return err
	}

	return nil
}

func (fm *FluxMonitor) statsAndStatusForRound(roundID uint32, newRoundLogs uint) (FluxMonitorRoundStatsV2, pipeline.RunStatus, error) {
	roundStats, err := fm.orm.FindOrCreateFluxMonitorRoundStats(fm.contractAddress, roundID, newRoundLogs)
	if err != nil {
		return FluxMonitorRoundStatsV2{}, pipeline.RunStatusUnknown, err
	}

	// JobRun will not exist if this is the first time responding to this round
	var run pipeline.Run
	if roundStats.PipelineRunID.Valid {
		run, err = fm.pipelineORM.FindRun(roundStats.PipelineRunID.Int64)
		if err != nil {
			return FluxMonitorRoundStatsV2{}, pipeline.RunStatusUnknown, err
		}
	}

	return roundStats, run.Status(), nil
}
