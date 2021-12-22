package test

import (
	"PhoenixOracle/gophoenix/core/services"
	"PhoenixOracle/gophoenix/core/store/models"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestLoadingSavedSchedules(t *testing.T) {
	t.Parallel()
	RegisterTestingT(t)
	store := NewStore()
	defer CleanUpStore(store)

	j := NewJob()
	j.Initiators = []models.Initiator{{Type: "cron", Schedule: "* * * * *"}}
	jobWoCron := models.NewJob()
	assert.Nil(t, store.SaveJob(j))
	assert.Nil(t, store.SaveJob(jobWoCron))

	sched := services.NewScheduler(store)
	_ = sched.Start()


	jobRuns := []models.JobRun{}
	Eventually(func() []models.JobRun {
		store.Where("JobID", j.ID, &jobRuns)
		return jobRuns
	}).Should(HaveLen(1))

	sched.Stop()
}

func TestSchedulesWithEmptyCron(t *testing.T) {
	RegisterTestingT(t)
	store := NewStore()
	defer CleanUpStore(store)

	j := models.NewJob()
	_ = store.Save(&j)

	sched := services.NewScheduler(store)
	_ = sched.Start()
	defer sched.Stop()

	jobRuns := []models.JobRun{}
	Eventually(func() []models.JobRun {
		_ = store.Where("JobID", j.ID, &jobRuns)
		return jobRuns
	}).Should(HaveLen(0))
}

func TestAddJob(t *testing.T) {
	t.Parallel()
	RegisterTestingT(t)
	store := NewStore()
	sched := services.NewScheduler(store)
	sched.Start()
	defer CleanUpStore(store)
	defer sched.Stop()

	j := NewJobWithSchedule("* * * * *")
	err := store.SaveJob(j)
	assert.Nil(t, err)
	sched.AddJob(j)

	jobRuns := []models.JobRun{}
	Eventually(func() []models.JobRun {
		err = store.Where("JobID", j.ID, &jobRuns)
		assert.Nil(t, err)
		return jobRuns
	}).Should(HaveLen(1))
}

func TestAddJobWhenStopped(t *testing.T) {
	t.Parallel()
	RegisterTestingT(t)
	store := NewStore()
	defer CleanUpStore(store)
	sched := services.NewScheduler(store)

	defer sched.Stop()

	j := NewJobWithSchedule("* * * * *")
	assert.Nil(t, store.SaveJob(j))
	sched.AddJob(j)

	jobRuns := []models.JobRun{}
	Consistently(func() []models.JobRun {
		store.Where("JobID", j.ID, &jobRuns)
		return jobRuns
	}).Should(HaveLen(0))

	assert.Nil(t, sched.Start())
	Eventually(func() []models.JobRun {
		_ = store.Where("JobID", j.ID, &jobRuns)
		return jobRuns
	}).Should(HaveLen(1))
}

func TestOneTimeRunJobAt(t *testing.T) {
	RegisterTestingT(t)
	t.Parallel()

	store := NewStore()
	defer CleanUpStore(store)

	ot := services.OneTime{
		Clock: &NeverClock{},
		Store: store,
	}
	ot.Start()
	j := NewJob()
	assert.Nil(t, store.SaveJob(j))

	var finished bool
	go func() {
		ot.RunJobAt(models.Time{time.Now().Add(time.Hour)}, j)
		finished = true
	}()

	ot.Stop()

	Eventually(func() bool {
		return finished
	}).Should(Equal(true))
	jobRuns := []models.JobRun{}
	assert.Nil(t, store.Where("JobID", j.ID, &jobRuns))
	assert.Equal(t, 0, len(jobRuns))
}
