package utils

import (
	"time"

	"github.com/pkg/errors"
)

type SleeperTask interface {
	Stop() error
	WakeUp()
	WakeUpIfStarted()
}

type Worker interface {
	Work()
}

type sleeperTask struct {
	worker  Worker
	chQueue chan struct{}
	chStop  chan struct{}
	chDone  chan struct{}
	StartStopOnce
}

func NewSleeperTask(worker Worker) SleeperTask {
	s := &sleeperTask{
		worker:  worker,
		chQueue: make(chan struct{}, 1),
		chStop:  make(chan struct{}),
		chDone:  make(chan struct{}),
	}

	_ = s.StartOnce("Sleeper task", func() error {
		go s.workerLoop()
		return nil
	})

	return s
}

func (s *sleeperTask) Stop() error {
	return s.StopOnce("Sleeper task", func() error {
		close(s.chStop)
		select {
		case <-s.chDone:
		case <-time.After(15 * time.Second):
			return errors.New("Sleeper task took too long to stop")
		}
		return nil
	})
}

func (s *sleeperTask) WakeUpIfStarted() {
	s.IfStarted(func() {
		select {
		case s.chQueue <- struct{}{}:
		default:
		}
	})
}

func (s *sleeperTask) WakeUp() {
	if s.StartStopOnce.State() == StartStopOnce_Stopped {
		panic("cannot wake up stopped sleeper task")
	}
	select {
	case s.chQueue <- struct{}{}:
	default:
	}
}

func (s *sleeperTask) workerLoop() {
	defer close(s.chDone)

	for {
		select {
		case <-s.chQueue:
			s.worker.Work()
		case <-s.chStop:
			return
		}
	}
}

type SleeperTaskFuncWorker func()

func (fn SleeperTaskFuncWorker) Work() { fn() }
