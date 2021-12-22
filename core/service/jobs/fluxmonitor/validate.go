package fluxmonitor

import (
	"time"

	uuid "github.com/satori/go.uuid"

	"PhoenixOracle/core/service/job"
	"PhoenixOracle/db/models"
	"PhoenixOracle/util"
	"github.com/pelletier/go-toml"
	"github.com/pkg/errors"
)

type ValidationConfig interface {
	DefaultHTTPTimeout() models.Duration
}

func ValidatedFluxMonitorSpec(config ValidationConfig, ts string) (job.Job, error) {
	var jb = job.Job{
		ExternalJobID: uuid.NewV4(),
	}
	var spec job.FluxMonitorSpec
	tree, err := toml.Load(ts)
	if err != nil {
		return jb, err
	}
	err = tree.Unmarshal(&jb)
	if err != nil {
		return jb, err
	}
	err = tree.Unmarshal(&spec)
	if err != nil {
		var specIntThreshold job.FluxMonitorSpecIntThreshold
		err = tree.Unmarshal(&specIntThreshold)
		if err != nil {
			return jb, err
		}
		spec = job.FluxMonitorSpec{
			ContractAddress:     specIntThreshold.ContractAddress,
			Threshold:           float32(specIntThreshold.Threshold),
			AbsoluteThreshold:   float32(specIntThreshold.AbsoluteThreshold),
			PollTimerPeriod:     specIntThreshold.PollTimerPeriod,
			PollTimerDisabled:   specIntThreshold.PollTimerDisabled,
			IdleTimerPeriod:     specIntThreshold.IdleTimerPeriod,
			IdleTimerDisabled:   specIntThreshold.IdleTimerDisabled,
			DrumbeatSchedule:    specIntThreshold.DrumbeatSchedule,
			DrumbeatRandomDelay: specIntThreshold.DrumbeatRandomDelay,
			DrumbeatEnabled:     specIntThreshold.DrumbeatEnabled,
			MinPayment:          specIntThreshold.MinPayment,
		}
	}
	jb.FluxMonitorSpec = &spec

	if jb.Type != job.FluxMonitor {
		return jb, errors.Errorf("unsupported type %s", jb.Type)
	}

	minTaskTimeout, aTimeoutSet, err := jb.Pipeline.MinTimeout()
	if err != nil {
		return jb, err
	}
	timeouts := []time.Duration{
		config.DefaultHTTPTimeout().Duration(),
		time.Duration(jb.MaxTaskDuration),
	}
	if aTimeoutSet {
		timeouts = append(timeouts, minTaskTimeout)
	}
	var minTimeout time.Duration = 1<<63 - 1
	for _, timeout := range timeouts {
		if timeout < minTimeout {
			minTimeout = timeout
		}
	}

	if jb.FluxMonitorSpec.DrumbeatEnabled {
		err := utils.ValidateCronSchedule(jb.FluxMonitorSpec.DrumbeatSchedule)
		if err != nil {
			return jb, errors.Wrap(err, "while validating drumbeat schedule")
		}

		if !spec.IdleTimerDisabled {
			return jb, errors.Errorf("When the drumbeat ticker is enabled, the idle timer must be disabled. Please set IdleTimerDisabled to true")
		}
	}

	if !validatePollTimer(jb.FluxMonitorSpec.PollTimerDisabled, minTimeout, jb.FluxMonitorSpec.PollTimerPeriod) {
		return jb, errors.Errorf("pollTimer.period must be equal or greater than %v, got %v", minTimeout, jb.FluxMonitorSpec.PollTimerPeriod)
	}

	return jb, nil
}

func validatePollTimer(disabled bool, minTimeout time.Duration, period time.Duration) bool {
	if disabled {
		return true
	}

	return period >= minTimeout
}
