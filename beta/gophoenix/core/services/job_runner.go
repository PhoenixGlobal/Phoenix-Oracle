package services

import (
	"PhoenixOracle/gophoenix/core/adapters"
	"PhoenixOracle/gophoenix/core/logger"
	"PhoenixOracle/gophoenix/core/store"
	"PhoenixOracle/gophoenix/core/store/models"
	"fmt"
)

func StartJob(run models.JobRun, store *store.Store) (models.JobRun, error) {
	run.Status = "in progress"
	if err := store.Save(&run); err != nil {
		return run, runJobError(run, err)
	}

	logger.GetLogger().Infow("starting job", run.ForLogger()...)
	unfinished := run.UnfinishedTaskRuns()
	offset := len(run.TaskRuns) - len(unfinished)
	prevRun := run.NextTaskRun()
	for i, taskRun := range unfinished {
		prevRun = startTask(taskRun, prevRun.Result, store)
		run.TaskRuns[i+offset] = prevRun

		err:= store.Save(&run); if err != nil {
			fmt.Println("*****************************")
			return run, runJobError(run, err)
		}
		if prevRun.Result.Pending {
			logger.Infow("Task pending", run.ForLogger("task", i, "result", prevRun.Result)...)
			break
		} else {
			logger.Infow("Task finished", run.ForLogger("task", i, "result", prevRun.Result)...)
			if prevRun.Result.HasError() {
				break
			}
		}
	}

	run.Result = prevRun.Result
	if run.Result.HasError() {
		fmt.Println(run.Result.ErrorMessage)
		run.Status = "errored"
	} else if run.Result.Pending {
		run.Status = "pending"
	} else {
		run.Status = "completed"
	}

	return run, runJobError(run, store.Save(&run))
}

func startTask(run models.TaskRun, input models.RunResult,store *store.Store) models.TaskRun {
	run.Status = "in progress"
	adapter, err := adapters.For(run.Task)

	if err != nil {
		run.Status = "errored"
		run.Result.SetError(err)
		return run
	}
	run.Result = adapter.Perform(input, store)

	if run.Result.HasError() {
		run.Status = "errored"
	} else if run.Result.Pending {
		run.Status = "pending"
	} else {
		run.Status = "completed"
	}

	return run
}

func runJobError(run models.JobRun, err error) error {
	if err != nil {
		return fmt.Errorf("StartJob#%v: %v", run.JobID, err)
	}
	return nil
}