package use_cases

import "time"

type Clock interface {
	NowUTC() time.Time
}

type systemClock struct{}

func NewSystemClock() Clock {
	return systemClock{}
}

func (systemClock) NowUTC() time.Time {
	return time.Now().UTC()
}
