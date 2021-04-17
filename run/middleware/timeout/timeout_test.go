package timeout

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/fnrun/fnrun/fn"
	"github.com/fnrun/fnrun/run/config"
)

func TestNew_withTimeout(t *testing.T) {
	m := NewWithDuration(1 * time.Millisecond)
	f := NewSleepyFn(30 * time.Second)

	_, err := m.Invoke(context.Background(), "some input", f)

	if err != ErrContextDone {
		t.Errorf("expected err to be ErrContextDone but was: %#v", err)
	}
}

func TestNew(t *testing.T) {
	m := New()
	f := NewSleepyFn(1 * time.Microsecond)

	output, err := m.Invoke(context.Background(), "some input", f)
	if err != nil {
		t.Errorf("Invoke returned err: %#v", err)
	}

	want := "done"
	got := output.(string)

	if got != want {
		t.Errorf("want %q; got %q", want, got)
	}
}

func TestConfigureString(t *testing.T) {
	m := New()
	tm := m.(*timeoutMiddleware)
	if err := tm.ConfigureString("15s"); err != nil {
		t.Fatalf("ConfigureString returned err: %#v", err)
	}
	if tm.duration != 15*time.Second {
		t.Errorf("Duration contains incorrect duration; want 15s, got %s", tm.duration)
	}
}

func TestConfigureString_invalidString(t *testing.T) {
	m := New()
	err := config.Configure(m, "some invalid string")
	if err == nil {
		t.Error("expected config.Configure to return an error but it did not")
	}
}

// -----------------------------------------------------------------------------
// Sample function

var ErrContextDone = errors.New("context done before operation was completed")

type sleepyFn struct {
	duration time.Duration
}

func (s *sleepyFn) Invoke(ctx context.Context, input interface{}) (interface{}, error) {
	select {
	case <-ctx.Done():
		return nil, ErrContextDone
	case <-time.After(s.duration):
		return "done", nil
	}
}

func NewSleepyFn(sleepDuration time.Duration) fn.Fn {
	return &sleepyFn{
		duration: sleepDuration,
	}
}
