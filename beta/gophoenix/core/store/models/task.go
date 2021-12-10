package models

import (
	"encoding/json"
)

type Task struct {
	Type   string          `json:"type" storm:"index"`
	Params json.RawMessage `json:"params,omitempty"`
}

type TaskRun struct {
	ID string `storm:"id,index,unique"`
	Task
	Status string
	Result RunResult
}

func (self TaskRun) Completed() bool {
	return self.Status == "completed"
}

func (self TaskRun) Errored() bool {
	return self.Status == "errored"
}

