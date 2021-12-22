package timer

import (
	"github.com/pkg/errors"

	"PhoenixOracle/core/service/job"
	"PhoenixOracle/core/service/pipeline"
)

type Delegate struct {
	pipelineRunner pipeline.Runner
}

var _ job.Delegate = (*Delegate)(nil)

func NewDelegate(pipelineRunner pipeline.Runner) *Delegate {
	return &Delegate{
		pipelineRunner: pipelineRunner,
	}
}

func (d *Delegate) JobType() job.Type {
	return job.Cron
}

func (Delegate) AfterJobCreated(spec job.Job)  {}
func (Delegate) BeforeJobDeleted(spec job.Job) {}

func (d *Delegate) ServicesForSpec(spec job.Job) (services []job.Service, err error) {
	spec.PipelineSpec.JobName = spec.Name.ValueOrZero()
	spec.PipelineSpec.JobID = spec.ID

	if spec.CronSpec == nil {
		return nil, errors.Errorf("services.Delegate expects a *jobSpec.CronSpec to be present, got %v", spec)
	}

	cron, err := NewCronFromJobSpec(spec, d.pipelineRunner)
	if err != nil {
		return nil, err
	}

	return []job.Service{cron}, nil
}
