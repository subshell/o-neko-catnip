package utils

import (
	"sync"
	"time"
)

type Memoized[T any] struct {
	value      T
	hasValue   bool
	generator  func() (T, error)
	validUntil time.Time
	memoizeFor time.Duration
	mutex      sync.Mutex
	clock      Clock
}

func Memoize[T any](memoizeFor time.Duration, supplier func() (T, error)) *Memoized[T] {
	return memoizeWithClock(memoizeFor, supplier, newClock())
}

func memoizeWithClock[T any](memoizeFor time.Duration, supplier func() (T, error), clock Clock) *Memoized[T] {
	return &Memoized[T]{
		generator:  supplier,
		memoizeFor: memoizeFor,
		hasValue:   false,
		clock:      clock,
	}
}

func (m *Memoized[T]) Get() (T, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.hasValue && m.validUntil.After(m.clock.Now()) {
		return m.value, nil
	}

	value, err := m.generator()

	if err != nil {
		return m.value, err
	}

	m.value = value
	m.validUntil = m.clock.Now().Add(m.memoizeFor)
	m.hasValue = true
	return m.value, nil
}
