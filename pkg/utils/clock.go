package utils

import "time"

type Clock interface {
	Now() time.Time
}

type DefaultClock struct {
}

func (d *DefaultClock) Now() time.Time {
	return time.Now()
}

func newClock() Clock {
	return &DefaultClock{}
}

type TimeMachine struct {
	currentNow time.Time
}

func newTimeMachine() *TimeMachine {
	return &TimeMachine{
		currentNow: time.Now(),
	}
}

func (m *TimeMachine) Now() time.Time {
	return m.currentNow
}

func (m *TimeMachine) ResetToRealNow() {
	m.currentNow = time.Now()
}

func (m *TimeMachine) TimeTravel(d time.Duration) {
	m.currentNow = m.currentNow.Add(d)
}
