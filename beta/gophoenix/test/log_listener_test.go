package test

import (
	"PhoenixOracle/gophoenix/core/services"
	strpkg "PhoenixOracle/gophoenix/core/store"
	"PhoenixOracle/gophoenix/core/store/models"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestLogListenerStart(t *testing.T) {
	t.Parallel()

	store := NewStore()
	defer CleanUpStore(store)
	eth := MockEthOnStore(store)
	ll := services.LogListener{Store: store}
	defer ll.Stop()

	assert.Nil(t, store.SaveJob(NewJobWithLogInitiator()))
	assert.Nil(t, store.SaveJob(NewJobWithLogInitiator()))
	eth.RegisterSubscription("logs", make(chan strpkg.EventLog))
	eth.RegisterSubscription("logs", make(chan strpkg.EventLog))

	ll.Start()

	assert.True(t, eth.AllCalled())
}

func TestLogListenerAddJob(t *testing.T) {
	t.Parallel()
	RegisterTestingT(t)

	store := NewStore()
	defer CleanUpStore(store)
	eth := MockEthOnStore(store)
	ll := services.LogListener{Store: store}
	defer ll.Stop()
	ll.Start()

	j := NewJobWithLogInitiator()
	assert.Nil(t, store.SaveJob(j))
	logChan := make(chan strpkg.EventLog, 1)
	initr := j.Initiators[0]
	eth.RegisterSubscription("logs", logChan)

	ll.AddJob(j)

	logChan <- strpkg.EventLog{Address: initr.Address}
	jobRuns := []models.JobRun{}
	Eventually(func() []models.JobRun {
		store.Where("JobID", j.ID, &jobRuns)
		return jobRuns
	}).Should(HaveLen(1))

	assert.True(t, eth.AllCalled())
}
