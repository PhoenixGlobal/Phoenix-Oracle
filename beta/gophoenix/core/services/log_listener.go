package services

import (
	"PhoenixOracle/gophoenix/core/store"
	"PhoenixOracle/gophoenix/core/store/models"
)

type LogListener struct {
	Store     *store.Store
}

func NewLogListener(store *store.Store) *LogListener {
	return &LogListener{
	}
}

func (self *LogListener) Start() error {
	return nil
}

func (self *LogListener) Stop() error {
	return nil
}

func (self *LogListener) AddJob(job models.Job) error {
	return nil
}