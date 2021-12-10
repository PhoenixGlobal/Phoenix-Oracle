package fluxmonitor

import (
	"PhoenixOracle/core/keystore"
	"PhoenixOracle/core/log"
	"PhoenixOracle/core/service/ethereum"
	"PhoenixOracle/core/service/job"
	"PhoenixOracle/core/service/pipeline"
	"PhoenixOracle/core/service/txmanager"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

type Delegate struct {
	db             *gorm.DB
	txm            transmitter
	ethKeyStore    keystore.Eth
	jobORM         job.ORM
	pipelineORM    pipeline.ORM
	pipelineRunner pipeline.Runner
	ethClient      ethereum.Client
	logBroadcaster log.Broadcaster
	cfg            Config
}

var _ job.Delegate = (*Delegate)(nil)

func NewDelegate(
	txm transmitter,
	ethKeyStore keystore.Eth,
	jobORM job.ORM,
	pipelineORM pipeline.ORM,
	pipelineRunner pipeline.Runner,
	db *gorm.DB,
	ethClient ethereum.Client,
	logBroadcaster log.Broadcaster,
	cfg Config,
) *Delegate {
	return &Delegate{
		db,
		txm,
		ethKeyStore,
		jobORM,
		pipelineORM,
		pipelineRunner,
		ethClient,
		logBroadcaster,
		cfg,
	}
}

func (d *Delegate) JobType() job.Type {
	return job.FluxMonitor
}

func (Delegate) AfterJobCreated(spec job.Job)  {}
func (Delegate) BeforeJobDeleted(spec job.Job) {}

func (d *Delegate) ServicesForSpec(spec job.Job) (services []job.Service, err error) {
	if spec.FluxMonitorSpec == nil {
		return nil, errors.Errorf("Delegate expects a *job.FluxMonitorSpec to be present, got %v", spec)
	}

	strategy := txmanager.NewQueueingTxStrategy(spec.ExternalJobID, d.cfg.FMDefaultTransactionQueueDepth)

	fm, err := NewFromJobSpec(
		spec,
		d.db,
		NewORM(d.db, d.txm, strategy),
		d.jobORM,
		d.pipelineORM,
		NewKeyStore(d.ethKeyStore),
		d.ethClient,
		d.logBroadcaster,
		d.pipelineRunner,
		d.cfg,
	)
	if err != nil {
		return nil, err
	}

	return []job.Service{fm}, nil
}
