package adapters

import (
	"PhoenixOracle/gophoenix/core/store"
	"PhoenixOracle/gophoenix/core/store/models"
)

type NoOp struct {
}

func (self *NoOp) Perform(input models.RunResult, _ *store.Store) models.RunResult {
	return models.RunResult{}
}

type NoOpPend struct{}

func (self *NoOpPend) Perform(input models.RunResult, _ *store.Store) models.RunResult {
	return models.RunResultPending(input)
}
