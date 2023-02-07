package utils

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"os"
	"sync"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

func Test_Set(t *testing.T) {
	set := NewSet[string]()

	set.Add("foo")
	set.Add("foo")
	set.Add("foo")
	set.Add("bar")
	set.Add("foo")
	set.Add("baz")

	assert.Equal(t, 3, set.Size())
	assert.True(t, set.Contains("foo"))
	assert.True(t, set.Contains("bar"))
	assert.True(t, set.Contains("baz"))

	assert.Len(t, set.Items(), 3)
}

func Test_Memoize_MemoizesValue(t *testing.T) {
	callCount := 0
	generator := func() (string, error) {
		callCount++
		return "foo", nil
	}

	m := Memoize(10*time.Second, generator)

	get, err := m.Get()
	assert.NoError(t, err)
	assert.Equal(t, "foo", get)

	get, err = m.Get()
	assert.NoError(t, err)
	assert.Equal(t, "foo", get)

	get, err = m.Get()
	assert.NoError(t, err)
	assert.Equal(t, "foo", get)

	assert.Equal(t, 1, callCount)
}

func Test_Memoize_ExpiresAfterTime(t *testing.T) {
	callCount := 0
	generator := func() (string, error) {
		callCount++
		return "foo", nil
	}

	tm := newTimeMachine()

	m := memoizeWithClock(10 * time.Second, generator, tm)

	get, err := m.Get()
	assert.NoError(t, err)
	assert.Equal(t, "foo", get)

	tm.TimeTravel(11 * time.Second)

	get, err = m.Get()
	assert.NoError(t, err)
	assert.Equal(t, "foo", get)

	tm.TimeTravel(11 * time.Second)

	get, err = m.Get()
	assert.NoError(t, err)
	assert.Equal(t, "foo", get)

	assert.Equal(t, 3, callCount)
}

func Test_Memoize_Concurrent(t *testing.T) {
	callCount := 0
	generator := func() (string, error) {
		callCount++
		return "foo", nil
	}

	m := Memoize(60*time.Second, generator)

	var wg sync.WaitGroup
	wg.Add(10)

	for i := 0; i < 10; i++ {
		go func() {
			get, err := m.Get()
			assert.NoError(t, err)
			assert.Equal(t, "foo", get)
			wg.Done()
		}()
	}

	wg.Wait()

	assert.Equal(t, 1, callCount)
}

func Test_Memoize_Error(t *testing.T) {
	generator := func() (string, error) {
		return "", fmt.Errorf("oops")
	}

	m := Memoize(10 * time.Second, generator)

	result, err := m.Get()

	assert.Empty(t, result)
	assert.Errorf(t, err, "oops")
}

func TestTimeMachine(t *testing.T) {
	tm := newTimeMachine()
	start := tm.Now()
	assert.True(t, start.Before(time.Now()), "Start time should be before now")

	tm.TimeTravel(1 * time.Hour)
	assert.True(t, start.Before(tm.Now()), "TimeTravel should have moved the time forwards")

	tm.TimeTravel(-2 * time.Hour)
	assert.True(t, start.After(tm.Now()), "TimeTravel should have moved the time backwards")

	tm.ResetToRealNow()
	assert.True(t, start.Before(tm.Now()), "ResetToRealNow should have moved the time to the current time")
}
