package models

import (
	"encoding/json"
	"fmt"
	"github.com/araddon/dateparse"
	"github.com/ethereum/go-ethereum/common"
	"github.com/mrwonko/cron"
	uuid "github.com/satori/go.uuid"
	"time"
)

type Job struct {
	ID        string    `storm:"id,index,unique"`
	Initiators []Initiator `json:"initiators"`
	Tasks     []Task    `json:"tasks" storm:"inline"`
	CreatedAt Time `storm:"index"`
}

func NewJob() Job {
	return Job{ID: uuid.NewV4().String(), CreatedAt: Time{Time: time.Now()}}
}

func (self Job) NewRun() JobRun {
	taskRuns := make([]TaskRun, len(self.Tasks))
	for i, task := range self.Tasks {
		taskRuns[i] = TaskRun{
			ID:   uuid.NewV4().String(),
			Task: task,
		}
	}
	run := JobRun{
		ID:        uuid.NewV4().String(),
		JobID:     self.ID,
		CreatedAt: time.Now(),
		TaskRuns:  taskRuns,
	}

	return run
}

func (self Job) InitiatorsFor(t string) []Initiator {
	list := []Initiator{}
	for _, initr := range self.Initiators {
		if initr.Type == t {
			list = append(list, initr)
		}
	}
	return list
}

func (self Job) WebAuthorized() bool {
	for _, initr := range self.Initiators {
		if initr.Type == "web" {
			return true
		}
	}
	return false
}

type Initiator struct {
	ID       int    `storm:"id,increment"`
	JobID    string `storm:"index"`
	Type     string `json:"type" storm:"index"`
	Schedule Cron   `json:"schedule,omitempty"`
	Time     Time   `json:"time,omitempty"`
	Ran      bool   `json:"ranAt,omitempty"`
	Address  common.Address `json:"address,omitempty" storm:"index"`
}

type Time struct {
	time.Time
}
func (self *Time) UnmarshalJSON(b []byte) error {
	var s string
	err := json.Unmarshal(b, &s)
	t, err := dateparse.ParseAny(s)
	self.Time = t
	return err
}

func (self *Time) ISO8601() string {
	return self.UTC().Format("2006-01-02T15:04:05Z07:00")
}

func (self *Time) DurationFromNow() time.Duration {
	return self.Time.Sub(time.Now())
}

type Cron string

func (self *Cron) UnmarshalJSON(b []byte) error{
	var s string
	err := json.Unmarshal(b, &s)
	if err != nil{
		return fmt.Errorf("Cron: %v",err)
	}
	if s == "" {
		return nil
	}
	_, err = cron.Parse(s)

	if err != nil{
		return fmt.Errorf("Cron: %v", err)
	}
	*self = Cron(s)
	return nil
}
