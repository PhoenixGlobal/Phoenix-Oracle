package test

import (
	"PhoenixOracle/gophoenix/core/adapters"
	"PhoenixOracle/gophoenix/core/store/models"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestSave(t *testing.T) {
	t.Parallel()
	store := NewStore()
	defer store.Close()
	j1  := NewJobWithSchedule("* * * * *")
	store.Save(&j1)

	var j2 models.Job
	store.One("ID",j1.ID,&j2)

	assert.Equal(t, j1.Initiators[0].Schedule, j2.Initiators[0].Schedule)
}

func TestJobNewRun(t *testing.T) {
	t.Parallel()
	store := NewStore()
	defer store.Close()

	job := NewJobWithSchedule("1 * * * *")
	job.Tasks = []models.Task{models.Task{Type: "NoOp"}}

	newRun := job.NewRun()
	assert.Equal(t, job.ID, newRun.JobID)
	assert.Equal(t, 1, len(newRun.TaskRuns))
	assert.Equal(t, "NoOp", job.Tasks[0].Type)
	assert.Nil(t, job.Tasks[0].Params)
	adapter, _ := adapters.For(job.Tasks[0])
	assert.NotNil(t, adapter)

}

func TestTimeDurationFromNow(t *testing.T) {
	future := models.Time{time.Now().Add(time.Second)}
	duration := future.DurationFromNow()
	assert.True(t, 0 < duration)
}
