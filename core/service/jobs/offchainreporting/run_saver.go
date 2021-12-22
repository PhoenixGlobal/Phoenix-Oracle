package offchainreporting

import (
	"PhoenixOracle/lib/gracefulpanic"
	"github.com/smartcontractkit/sqlx"

	"PhoenixOracle/core/service/pipeline"
	"PhoenixOracle/lib/logger"
	"PhoenixOracle/util"
)

type RunResultSaver struct {
	utils.StartStopOnce

	db             *sqlx.DB
	runResults     <-chan pipeline.Run
	pipelineRunner pipeline.Runner
	done           chan struct{}
	logger         logger.Logger
}

func NewResultRunSaver(db *sqlx.DB, runResults <-chan pipeline.Run, pipelineRunner pipeline.Runner, done chan struct{},
	logger logger.Logger,
) *RunResultSaver {
	return &RunResultSaver{
		db:             db,
		runResults:     runResults,
		pipelineRunner: pipelineRunner,
		done:           done,
		logger:         logger,
	}
}

func (r *RunResultSaver) Start() error {
	return r.StartOnce("RunResultSaver", func() error {
		go gracefulpanic.WrapRecover(func() {
			for {
				select {
				case run := <-r.runResults:
					r.logger.Infow("RunSaver: saving job run", "run", run)

					_, err := r.pipelineRunner.InsertFinishedRun(r.db, run, false)
					if err != nil {
						r.logger.Errorw("error inserting finished results", "err", err)
					}
				case <-r.done:
					return
				}
			}
		})
		return nil
	})
}

func (r *RunResultSaver) Close() error {
	return r.StopOnce("RunResultSaver", func() error {
		r.done <- struct{}{}

		for {
			select {
			case run := <-r.runResults:
				r.logger.Infow("RunSaver: saving job run before exiting", "run", run, "task results")
				_, err := r.pipelineRunner.InsertFinishedRun(r.db, run, false)
				if err != nil {
					r.logger.Errorw("error inserting finished results", "err", err)
				}
			default:
				return nil
			}
		}
	})
}
