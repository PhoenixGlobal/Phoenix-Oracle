package test

import (
	"PhoenixOracle/gophoenix/core/services"
	"PhoenixOracle/gophoenix/core/store/models"
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestRetrievingJobRunsWithErrorsFromDB(t *testing.T) {
	store := NewStore()
	defer CleanUpStore(store)

	job := models.NewJob()
	jr := job.NewRun()
	jr.Result = models.RunResultWithError(fmt.Errorf("bad idea"))
	err := store.Save(&jr)
	assert.Nil(t, err)

	run := models.JobRun{}
	err = store.One("ID", jr.ID, &run)
	assert.Nil(t, err)
	assert.True(t, run.Result.HasError())
	assert.Equal(t, "bad idea", run.Result.Error())
}


func TestJobTransitionToPending(t *testing.T) {
	t.Parallel()
	store := NewStore()
	defer CleanUpStore(store)

	job := models.NewJob()
	job.Tasks = []models.Task{models.Task{Type: "NoOpPend"}}

	run := job.NewRun()
	services.StartJob(run, store)

	store.One("ID", run.ID, &run)
	assert.Equal(t, "pending", run.Status)
}

func TestTaskRunsToRun(t *testing.T) {
	t.Parallel()
	store := NewStore()
	defer CleanUpStore(store)

	j := models.NewJob()
	j.Tasks = []models.Task{
		{Type: "NoOp"},
		{Type: "NoOpPend"},
		{Type: "NoOp"},
	}
	assert.Nil(t, store.SaveJob(j))
	jr := j.NewRun()
	assert.Equal(t, jr.TaskRuns, jr.UnfinishedTaskRuns())

	jr, err := services.StartJob(jr, store)
	assert.Nil(t, err)
	assert.Equal(t, jr.TaskRuns[1:], jr.UnfinishedTaskRuns())
}

