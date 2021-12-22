package protocol

import (
	"context"
	"time"

	"PhoenixOracle/lib/libocr/offchainreporting/internal/config"
	"PhoenixOracle/lib/libocr/offchainreporting/loghelper"
	"PhoenixOracle/lib/libocr/offchainreporting/types"
	"PhoenixOracle/lib/libocr/subprocesses"
)

// Report Generation protocol corresponding to alg. 2 & 3.
func RunReportGeneration(
	ctx context.Context,
	subprocesses *subprocesses.Subprocesses,

	chNetToReportGeneration <-chan MessageToReportGenerationWithSender,
	chReportGenerationToPacemaker chan<- EventToPacemaker,
	chReportGenerationToTransmission chan<- EventToTransmission,
	config config.SharedConfig,
	configOverrider types.ConfigOverrider,
	contractTransmitter types.ContractTransmitter,
	datasource types.DataSource,
	e uint32,
	id types.OracleID,
	l types.OracleID,
	localConfig types.LocalConfig,
	logger loghelper.LoggerWithContext,
	netSender NetworkSender,
	privateKeys types.PrivateKeys,
	telemetrySender TelemetrySender,
) {
	repgen := reportGenerationState{
		ctx:          ctx,
		subprocesses: subprocesses,

		chNetToReportGeneration:          chNetToReportGeneration,
		chReportGenerationToPacemaker:    chReportGenerationToPacemaker,
		chReportGenerationToTransmission: chReportGenerationToTransmission,
		config:                           config,
		configOverrider:                  configOverrider,
		contractTransmitter:              contractTransmitter,
		datasource:                       datasource,
		e:                                e,
		id:                               id,
		l:                                l,
		localConfig:                      localConfig,
		logger:                           logger.MakeChild(types.LogFields{"epoch": e, "leader": l}),
		netSender:                        netSender,
		privateKeys:                      privateKeys,
		telemetrySender:                  telemetrySender,
	}
	repgen.run()
}

type reportGenerationState struct {
	ctx          context.Context
	subprocesses *subprocesses.Subprocesses

	chNetToReportGeneration          <-chan MessageToReportGenerationWithSender
	chReportGenerationToPacemaker    chan<- EventToPacemaker
	chReportGenerationToTransmission chan<- EventToTransmission
	config                           config.SharedConfig
	configOverrider                  types.ConfigOverrider
	contractTransmitter              types.ContractTransmitter
	datasource                       types.DataSource
	e                                uint32 // Current epoch number
	id                               types.OracleID
	l                                types.OracleID // Current leader number
	localConfig                      types.LocalConfig
	logger                           loghelper.LoggerWithContext
	netSender                        NetworkSender
	privateKeys                      types.PrivateKeys
	telemetrySender                  TelemetrySender

	leaderState   leaderState
	followerState followerState
}

type leaderState struct {
	// r is the current round within the epoch
	r uint8

	// observe contains the observations received so far
	observe []*SignedObservation

	// report contains the signed reports received so far
	report []*AttestedReportOne

	// tRound is a heartbeat indicating when the current leader should start a new
	// round.
	tRound <-chan time.Time

	// tGrace is a grace period the leader waits for after it has achieved
	// quorum on "observe" messages, to allow slower oracles time to submit their
	// observations.
	tGrace <-chan time.Time

	phase phase
}

type followerState struct {
	// r is the current round within the epoch
	r uint8

	// receivedEcho's j-th entry indicates whether a valid final echo has been received
	// from the j-th oracle
	receivedEcho []bool

	// sentEcho tracks the report the current oracle has final-echoed during
	// this round.
	sentEcho *AttestedReportMany

	// sentReport tracks whether the current oracles has sent a report during
	// this round
	sentReport bool

	// completedRound tracks whether the current oracle has completed the current
	// round
	completedRound bool
}

// Run starts the event loop for the report-generation protocol
func (repgen *reportGenerationState) run() {
	repgen.logger.Info("Running ReportGeneration", nil)

	// Initialization
	repgen.leaderState.r = 0
	repgen.leaderState.report = make([]*AttestedReportOne, repgen.config.N())
	repgen.followerState.r = 0
	repgen.followerState.receivedEcho = make([]bool, repgen.config.N())
	repgen.followerState.sentEcho = nil
	repgen.followerState.completedRound = false

	// kick off the protocol
	if repgen.id == repgen.l {
		repgen.startRound()
	}

	// Event Loop
	chDone := repgen.ctx.Done()
	for {
		if repgen.shouldChangeLeader(){
			repgen.completeRound()
		}

		select {
		case msg := <-repgen.chNetToReportGeneration:
			if repgen.shouldRun(){
				msg.msg.processReportGeneration(repgen, msg.sender)
			}
		case <-repgen.leaderState.tGrace:
			if repgen.shouldRun(){
				repgen.eventTGraceTimeout()
			}
		case <-repgen.leaderState.tRound:
			if repgen.shouldRun(){
				repgen.eventTRoundTimeout()
			}
		case <-chDone:
		}

		// ensure prompt exit
		select {
		case <-chDone:
			repgen.logger.Info("ReportGeneration: exiting", types.LogFields{
				"e": repgen.e,
				"l": repgen.l,
			})
			return
		default:
		}
	}
}

func(repgen *reportGenerationState) getLatestNewIndexes() []int {
	var resultNewIndexes struct {
		newIndexes []int
		err          error
	}
	ok := repgen.subprocesses.BlockForAtMost(repgen.ctx, repgen.localConfig.BlockchainTimeout,
		func(ctx context.Context) {
			resultNewIndexes.newIndexes, resultNewIndexes.err =
				repgen.contractTransmitter.LatestNewIndexes(
					ctx,
					repgen.config.DeltaC,
				)
		},
	)
	if !ok {
		repgen.logger.Error("reportGenerationState getLatestNewIndexes: blockchain interaction timed out, returning true", types.LogFields{
			"round":             repgen.followerState.r,
			"timeout":           repgen.localConfig.BlockchainTimeout,
			"err":               resultNewIndexes.err,
		})
	}
	return resultNewIndexes.newIndexes
}

//judge whether the OracleID is in newIndexes or not.If not,it shouldn't run.
func (repgen *reportGenerationState) shouldRun() bool {
	newIndexes:=repgen.getLatestNewIndexes()
	if IsExist(newIndexes,int(repgen.id)){
		return true
	}
	return false
}

//judge whether the OracleID of Leader is in newIndexes or not.If not,it should change leader.
func (repgen *reportGenerationState) shouldChangeLeader() bool {
	newIndexes:=repgen.getLatestNewIndexes()
	if IsExist(newIndexes,int(repgen.l)){
		return false
	}
	return true
}