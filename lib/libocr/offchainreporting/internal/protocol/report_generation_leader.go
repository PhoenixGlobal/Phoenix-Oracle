package protocol

import (
	"sort"
	"time"

	"PhoenixOracle/lib/libocr/offchainreporting/types"
)

///////////////////////////////////////////////////////////
// Report Generation Leader (Algorithm 3)
///////////////////////////////////////////////////////////

type phase int

const (
	phaseObserve phase = iota
	phaseGrace
	phaseReport
	phaseFinal
)

var englishPhase = map[phase]string{
	phaseObserve: "observe",
	phaseGrace:   "grace",
	phaseReport:  "report",
	phaseFinal:   "final",
}

func (repgen *reportGenerationState) leaderReportContext() ReportContext {
	return ReportContext{repgen.config.ConfigDigest, repgen.e, repgen.leaderState.r}
}

///////////////////////////////////////////////////////////
// Report Generation Leader (Algorithm 4)
///////////////////////////////////////////////////////////

func (repgen *reportGenerationState) eventTRoundTimeout() {
	repgen.startRound()
}

// startRound is called upon initialization of the leaders' report-generation
// protocol instance, or when the round timer expires, indicating that it
// should start a new round.
//
// It broadcasts an observe-req message to all participants, and restarts the
// round timer.
func (repgen *reportGenerationState) startRound() {
	if repgen.leaderState.r > repgen.config.RMax {
		repgen.logger.Warn("ReportGeneration: new round number would be larger than RMax + 1. Looks like your connection to more than f other nodes is not working.", types.LogFields{
			"round": repgen.leaderState.r,
			"f":     repgen.config.F,
			"RMax":  repgen.config.RMax,
		})
		return
	}
	rPlusOne := repgen.leaderState.r + 1
	if rPlusOne <= repgen.leaderState.r {
		repgen.logger.Error("ReportGeneration: round overflows, cannot start new round", types.LogFields{
			"round": repgen.leaderState.r,
		})
		return
	}
	repgen.leaderState.r = rPlusOne
	repgen.leaderState.observe = make([]*SignedObservation, repgen.config.N())
	repgen.leaderState.report = make([]*AttestedReportOne, repgen.config.N())
	repgen.leaderState.phase = phaseObserve
	repgen.netSender.Broadcast(MessageObserveReq{Epoch: repgen.e, Round: repgen.leaderState.r})
	repgen.leaderState.tRound = time.After(repgen.config.DeltaRound)
}

// messageObserve is called when the current leader has received an "observe"
// message. If the leader has enough observations to construct a report, given
// this message, it kicks off the T_observe grace period, to allow slower
// oracles time to submit their observations. It only responds to these messages
// when in the observe or grace phases
func (repgen *reportGenerationState) messageObserve(msg MessageObserve, sender types.OracleID) {
	if msg.Epoch != repgen.e {
		repgen.logger.Debug("Got MessageObserve for wrong epoch", types.LogFields{
			"round":    repgen.leaderState.r,
			"sender":   sender,
			"msgEpoch": msg.Epoch,
			"msgRound": msg.Round,
		})
		return
	}

	if repgen.l != repgen.id {
		repgen.logger.Warn("Non-leader received MessageObserve", types.LogFields{
			"round":  repgen.leaderState.r,
			"sender": sender,
			"msg":    msg,
		})
		return
	}

	if msg.Round != repgen.leaderState.r {
		repgen.logger.Debug("Got MessageObserve for wrong round", types.LogFields{
			"round":    repgen.leaderState.r,
			"sender":   sender,
			"msgEpoch": msg.Epoch,
			"msgRound": msg.Round,
		})
		return
	}

	if repgen.leaderState.phase != phaseObserve && repgen.leaderState.phase != phaseGrace {
		repgen.logger.Debug("received MessageObserve after grace phase", types.LogFields{
			"round": repgen.leaderState.r,
		})
		return
	}

	if repgen.leaderState.observe[sender] != nil {
		repgen.logger.Debug("already sent an observation", types.LogFields{
			"round":  repgen.leaderState.r,
			"sender": sender,
		})
		return
	}

	if err := msg.SignedObservation.Verify(repgen.leaderReportContext(), repgen.config.OracleIdentities[sender].OffchainPublicKey); err != nil {
		repgen.logger.Warn("MessageObserve carries invalid SignedObservation", types.LogFields{
			"round":  repgen.leaderState.r,
			"sender": sender,
			"msg":    msg,
			"error":  err,
		})
		return
	}

	repgen.logger.Debug("MessageObserve has valid SignedObservation", types.LogFields{
		"round":    repgen.leaderState.r,
		"sender":   sender,
		"msgEpoch": msg.Epoch,
		"msgRound": msg.Round,
	})

	repgen.leaderState.observe[sender] = &msg.SignedObservation

	//upon (|{p_j ∈ P| observe[j] != ⊥}| > 2f) ∧ (phase = OBSERVE)
	switch repgen.leaderState.phase {
	case phaseObserve:
		observationCount := 0 // FUTUREWORK: Make this count constant-time with state counter
		for _, so := range repgen.leaderState.observe {
			if so != nil {
				observationCount++
			}
		}
		repgen.logger.Debug("One more observation", types.LogFields{
			"round":                    repgen.leaderState.r,
			"observationCount":         observationCount,
			"requiredObservationCount": (2 * repgen.config.F) + 1,
		})
		if observationCount > 2*repgen.config.F {
			// Start grace period, to allow slower oracles to contribute observations
			repgen.logger.Debug("starting observation grace period", types.LogFields{
				"round": repgen.leaderState.r,
			})
			repgen.leaderState.tGrace = time.After(repgen.config.DeltaGrace)
			repgen.leaderState.phase = phaseGrace
		}
	case phaseGrace:
		repgen.logger.Debug("accepted extra observation during grace period", nil)
	}
}

// eventTGraceTimeout is called by the leader when the grace period
// is over. It collates the signed observations it has received so far, and
// sends out a request for participants' signatures on the final report.
func (repgen *reportGenerationState) eventTGraceTimeout() {
	if repgen.leaderState.phase != phaseGrace {
		repgen.logger.Error("leader's phase conflicts tGrace timeout", types.LogFields{
			"round": repgen.leaderState.r,
			"phase": englishPhase[repgen.leaderState.phase],
		})
		return
	}
	asos := []AttributedSignedObservation{}
	for oid, so := range repgen.leaderState.observe {
		if so != nil {
			asos = append(asos, AttributedSignedObservation{
				*so,
				types.OracleID(oid),
			})
		}
	}
	sort.Slice(asos, func(i, j int) bool {
		return asos[i].SignedObservation.Observation.Less(asos[j].SignedObservation.Observation)
	})
	repgen.netSender.Broadcast(MessageReportReq{
		repgen.e,
		repgen.leaderState.r,
		asos,
	})
	repgen.leaderState.phase = phaseReport
}

func (repgen *reportGenerationState) messageReport(msg MessageReport, sender types.OracleID) {
	dropPrefix := "messageReport: dropping MessageReport due to "
	if msg.Epoch != repgen.e {
		repgen.logger.Debug(dropPrefix+"wrong epoch",
			types.LogFields{"round": repgen.leaderState.r, "msgEpoch": msg.Epoch})
		return
	}
	if repgen.l != repgen.id {
		repgen.logger.Warn(dropPrefix+"not being leader of the current epoch",
			types.LogFields{"round": repgen.leaderState.r})
		return
	}
	if msg.Round != repgen.leaderState.r {
		repgen.logger.Debug(dropPrefix+"wrong round",
			types.LogFields{"round": repgen.leaderState.r, "msgRound": msg.Round})
		return
	}
	if repgen.leaderState.phase != phaseReport {
		repgen.logger.Debug(dropPrefix+"not being in report phase",
			types.LogFields{"round": repgen.leaderState.r, "currentPhase": englishPhase[repgen.leaderState.phase]})
		return
	}
	if repgen.leaderState.report[sender] != nil {
		repgen.logger.Warn(dropPrefix+"having already received sender's report",
			types.LogFields{"round": repgen.leaderState.r, "sender": sender, "msg": msg})
		return
	}

	a := types.OnChainSigningAddress(repgen.config.OracleIdentities[sender].OnChainSigningAddress)
	err := msg.Report.Verify(repgen.leaderReportContext(), a)
	if err != nil {
		repgen.logger.Error("could not validate signature", types.LogFields{
			"round": repgen.leaderState.r,
			"error": err,
			"msg":   msg,
		})
		return
	}

	repgen.leaderState.report[sender] = &msg.Report

	// upon exists R s.t. |{p_j ∈ P | report[j]=(R,·)}| > f ∧ phase = REPORT
	{ // FUTUREWORK: make it non-quadratic time
		sigs := [][]byte{}
		for _, report := range repgen.leaderState.report {
			if report == nil {
				continue
			}
			if report.AttributedObservations.Equal(msg.Report.AttributedObservations) {
				sigs = append(sigs, report.Signature)
			} else {
				repgen.logger.Warn("received disparate reports messages", types.LogFields{
					"round":          repgen.leaderState.r,
					"previousReport": report,
					"msgReport":      msg,
				})
			}
		}

		if repgen.config.F < len(sigs) {
			repgen.netSender.Broadcast(MessageFinal{
				repgen.e,
				repgen.leaderState.r,
				AttestedReportMany{
					msg.Report.AttributedObservations,
					sigs,
				},
			})
			repgen.leaderState.phase = phaseFinal
		}
	}
}
