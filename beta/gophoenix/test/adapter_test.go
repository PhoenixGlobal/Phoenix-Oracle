package test

import (
	"PhoenixOracle/gophoenix/core/adapters"
	"PhoenixOracle/gophoenix/core/store/models"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCreatingAdapterWithConfig(t *testing.T) {
	store := NewStore()
	defer CleanUpStore(store)
	task := models.Task{Type: "NoOp"}
	adapter, err := adapters.For(task)
	adapter.Perform(models.RunResult{}, store)
	assert.Nil(t, err)
}