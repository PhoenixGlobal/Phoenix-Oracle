package utils

import "time"

type Nower interface {
	Now() time.Time
}

type Afterer interface {
	After(d time.Duration) <-chan time.Time
}

type AfterNower interface {
	After(d time.Duration) <-chan time.Time
	Now() time.Time
}

type Clock struct{}

func (Clock) Now() time.Time {
	return time.Now()
}

func (Clock) After(d time.Duration) <-chan time.Time {
	return time.After(d)
}
