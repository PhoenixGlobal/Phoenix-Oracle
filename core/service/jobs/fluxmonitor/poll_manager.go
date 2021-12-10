package fluxmonitor

import (
	"fmt"
	"time"

	"PhoenixOracle/internal/gethwrappers/generated/flux_aggregator_wrapper"
	"PhoenixOracle/lib/logger"
	"PhoenixOracle/util"
)

type PollManagerConfig struct {
	IsHibernating           bool
	PollTickerInterval      time.Duration
	PollTickerDisabled      bool
	IdleTimerPeriod         time.Duration
	IdleTimerDisabled       bool
	DrumbeatSchedule        string
	DrumbeatEnabled         bool
	DrumbeatRandomDelay     time.Duration
	HibernationPollPeriod   time.Duration
	MinRetryBackoffDuration time.Duration
	MaxRetryBackoffDuration time.Duration
}

type PollManager struct {
	cfg PollManagerConfig

	hibernationTimer utils.ResettableTimer
	pollTicker       utils.PausableTicker
	idleTimer        utils.ResettableTimer
	roundTimer       utils.ResettableTimer
	retryTicker      utils.BackoffTicker
	drumbeat         utils.CronTicker
	chPoll           chan PollRequest

	logger *logger.Logger
}

func NewPollManager(cfg PollManagerConfig, logger *logger.Logger) (*PollManager, error) {
	minBackoffDuration := cfg.MinRetryBackoffDuration
	if cfg.IdleTimerPeriod < minBackoffDuration {
		minBackoffDuration = cfg.IdleTimerPeriod
	}
	maxBackoffDuration := cfg.MaxRetryBackoffDuration
	if cfg.IdleTimerPeriod < maxBackoffDuration {
		maxBackoffDuration = cfg.IdleTimerPeriod
	}

	var idleTimer = utils.NewResettableTimer()
	if !cfg.IdleTimerDisabled {
		idleTimer.Reset(cfg.IdleTimerPeriod)
	}

	var drumbeatTicker utils.CronTicker
	var err error
	if cfg.DrumbeatEnabled {
		drumbeatTicker, err = utils.NewCronTicker(cfg.DrumbeatSchedule)
		if err != nil {
			return nil, err
		}
	}

	return &PollManager{
		cfg:    cfg,
		logger: logger,

		hibernationTimer: utils.NewResettableTimer(),
		pollTicker:       utils.NewPausableTicker(cfg.PollTickerInterval),
		idleTimer:        idleTimer,
		roundTimer:       utils.NewResettableTimer(),
		retryTicker:      utils.NewBackoffTicker(minBackoffDuration, maxBackoffDuration),
		drumbeat:         drumbeatTicker,
		chPoll:           make(chan PollRequest),
	}, nil
}

func (pm *PollManager) PollTickerTicks() <-chan time.Time {
	return pm.pollTicker.Ticks()
}

func (pm *PollManager) IdleTimerTicks() <-chan time.Time {
	return pm.idleTimer.Ticks()
}

func (pm *PollManager) HibernationTimerTicks() <-chan time.Time {
	return pm.hibernationTimer.Ticks()
}

func (pm *PollManager) RoundTimerTicks() <-chan time.Time {
	return pm.roundTimer.Ticks()
}

func (pm *PollManager) RetryTickerTicks() <-chan time.Time {
	return pm.retryTicker.Ticks()
}

func (pm *PollManager) DrumbeatTicks() <-chan time.Time {
	return pm.drumbeat.Ticks()
}

func (pm *PollManager) Poll() <-chan PollRequest {
	return pm.chPoll
}

func (pm *PollManager) Start(hibernate bool, roundState flux_aggregator_wrapper.OracleRoundState) {
	pm.cfg.IsHibernating = hibernate

	if pm.ShouldPerformInitialPoll() {
		go func() {
			select {
			case pm.chPoll <- PollRequest{PollRequestTypeInitial, time.Now()}:
			case <-time.After(5 * time.Second):
				pm.logger.Warn("Start up poll was not consumed")
			}
		}()
	}

	pm.maybeWarnAboutIdleAndPollIntervals()

	if hibernate {
		pm.Hibernate()
	} else {
		pm.Awaken(roundState)
	}
}

func (pm *PollManager) ShouldPerformInitialPoll() bool {
	return (!pm.cfg.PollTickerDisabled || !pm.cfg.IdleTimerDisabled) && !pm.cfg.IsHibernating
}


func (pm *PollManager) Reset(roundState flux_aggregator_wrapper.OracleRoundState) {
	if pm.cfg.IsHibernating {
		pm.hibernationTimer.Reset(pm.cfg.HibernationPollPeriod)
	} else {
		pm.startPollTicker()
		pm.startIdleTimer(roundState.StartedAt)
		pm.startRoundTimer(roundStateTimesOutAt(roundState))
		pm.startDrumbeat()
	}
}

func (pm *PollManager) ResetIdleTimer(roundStartedAtUTC uint64) {
	if !pm.cfg.IsHibernating {
		pm.startIdleTimer(roundStartedAtUTC)
	}
}

func (pm *PollManager) StartRetryTicker() bool {
	return pm.retryTicker.Start()
}

func (pm *PollManager) StopRetryTicker() {
	if pm.retryTicker.Stop() {
		pm.logger.Debug("stopped retry ticker")
	}
}

func (pm *PollManager) Stop() {
	pm.hibernationTimer.Stop()
	pm.pollTicker.Destroy()
	pm.idleTimer.Stop()
	pm.roundTimer.Stop()
	pm.drumbeat.Stop()
}

func (pm *PollManager) Hibernate() {
	pm.logger.Infof("entering hibernation mode (period: %v)", pm.cfg.HibernationPollPeriod)

	pm.cfg.IsHibernating = true
	pm.hibernationTimer.Reset(pm.cfg.HibernationPollPeriod)

	// Stop the other tickers
	pm.pollTicker.Pause()
	pm.idleTimer.Stop()
	pm.roundTimer.Stop()
	pm.drumbeat.Stop()
	pm.StopRetryTicker()
}

func (pm *PollManager) Awaken(roundState flux_aggregator_wrapper.OracleRoundState) {
	pm.logger.Info("exiting hibernation mode, reactivating contract")

	pm.cfg.IsHibernating = false
	pm.hibernationTimer.Stop()

	pm.startPollTicker()
	pm.startIdleTimer(roundState.StartedAt)
	pm.startRoundTimer(roundStateTimesOutAt(roundState))
	pm.startDrumbeat()
}

func (pm *PollManager) startPollTicker() {
	if pm.cfg.PollTickerDisabled {
		pm.pollTicker.Pause()

		return
	}

	pm.pollTicker.Resume()
}

func (pm *PollManager) startIdleTimer(roundStartedAtUTC uint64) {

	if pm.cfg.IdleTimerDisabled {
		pm.idleTimer.Stop()

		return
	}

	if roundStartedAtUTC == 0 {
		pm.logger.Debugw("not resetting idleTimer, no active round")

		return
	}

	startedAt := time.Unix(int64(roundStartedAtUTC), 0)
	deadline := startedAt.Add(pm.cfg.IdleTimerPeriod)
	deadlineDuration := time.Until(deadline)

	log := pm.logger.With(
		"pollFrequency", pm.cfg.PollTickerInterval,
		"idleDuration", pm.cfg.IdleTimerPeriod,
		"startedAt", roundStartedAtUTC,
		"timeUntilIdleDeadline", deadlineDuration,
	)

	if deadlineDuration <= 0 {
		log.Debugw("not resetting idleTimer, round was started further in the past than idle timer period")
		return
	}

	if pm.retryTicker.Stop() {
		pm.logger.Debugw("stopped the retryTicker")
	}

	pm.idleTimer.Reset(deadlineDuration)
	log.Debugw("resetting idleTimer")
}

func (pm *PollManager) startRoundTimer(roundTimesOutAt uint64) {
	log := pm.logger.With(
		"pollFrequency", pm.cfg.PollTickerInterval,
		"idleDuration", pm.cfg.IdleTimerPeriod,
		"timesOutAt", roundTimesOutAt,
	)

	if roundTimesOutAt == 0 {
		log.Debugw("disabling roundTimer, no active round")
		pm.roundTimer.Stop()

		return
	}

	timesOutAt := time.Unix(int64(roundTimesOutAt), 0)
	timeoutDuration := time.Until(timesOutAt)

	if timeoutDuration <= 0 {
		log.Debugw(fmt.Sprintf("disabling roundTimer, as the round is already past its timeout by %v", -timeoutDuration))
		pm.roundTimer.Stop()

		return
	}

	pm.roundTimer.Reset(timeoutDuration)
	log.Debugw("updating roundState.TimesOutAt", "value", roundTimesOutAt)
}

func (pm *PollManager) startDrumbeat() {
	if !pm.cfg.DrumbeatEnabled {
		if pm.drumbeat.Stop() {
			pm.logger.Debug("disabled drumbeat ticker")
		}
		return
	}

	if pm.drumbeat.Start() {
		pm.logger.Debugw("started drumbeat ticker", "schedule", pm.cfg.DrumbeatSchedule)
	}
}

func roundStateTimesOutAt(rs flux_aggregator_wrapper.OracleRoundState) uint64 {
	return rs.StartedAt + rs.Timeout
}

func (pm *PollManager) maybeWarnAboutIdleAndPollIntervals() {
	if !pm.cfg.IdleTimerDisabled && !pm.cfg.PollTickerDisabled && pm.cfg.IdleTimerPeriod < pm.cfg.PollTickerInterval {
		pm.logger.Warnw("The value of IdleTimerPeriod is lower than PollTickerInterval. The idle timer should usually be less frequent that poll",
			"IdleTimerPeriod", pm.cfg.IdleTimerPeriod, "PollTickerInterval", pm.cfg.PollTickerInterval)
	}
}
