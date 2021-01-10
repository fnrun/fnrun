package pool_test

import (
	"context"
	"fmt"
	"log"
	"testing"
	"time"

	"github.com/fnrun/fnrun/pkg/fn"
	"github.com/fnrun/fnrun/pkg/run"
	"github.com/fnrun/fnrun/pkg/run/config"
	"github.com/fnrun/fnrun/pkg/run/fn/pool"
)

func newPool(t *testing.T, template interface{}) fn.Fn {
	t.Helper()

	r := run.NewRegistry()
	r.RegisterFn("sleepy", NewSleepyFn)
	r.RegisterFn("echo", NewEchoFn)

	p := pool.New(r)
	err := config.Configure(p, map[string]interface{}{
		"maxWaitDuration": "5ms",
		"concurrency":     2,
		"template":        template,
	})
	if err != nil {
		t.Fatalf("Configuring the pool returned an err: %+v", err)
	}

	return p
}

func TestNew_echoFn(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	p := newPool(t, "echo")
	output, err := p.Invoke(ctx, "hi there")
	if err != nil {
		t.Errorf("Invoke returned error: %+v", err)
	}

	want := "echo: hi there"
	got := output.(string)

	if got != want {
		t.Errorf("output mismatch: want %q; got %q", want, got)
	}
}

func TestNew_withTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	p := newPool(t, "sleepy")

	go func() {
		p.Invoke(ctx, "first input")
	}()

	go func() {
		p.Invoke(ctx, "second input")
	}()

	// The following is cheating a little, but it is to get around a race
	// race condition with the kicking off of the go routines above.
	<-time.After(5 * time.Millisecond)

	// pool should be tapped now because the "sleepy" Fn sleeps for a second
	// before responding
	output, err := p.Invoke(ctx, "third input")

	if err != pool.ErrAvailabilityTimeout {
		t.Errorf("expected Invoke on tapped pool to return ErrAvailabilityTime but returned: %+v", err)
	}

	if output != nil {
		t.Errorf("expected output to be nil but it was: %#v", output)
	}
}

// -----------------------------------------------------------------------------
// Sample functions

type sleepyFn struct{}

func (*sleepyFn) Invoke(context.Context, interface{}) (interface{}, error) {
	log.Println("about to sleep")
	<-time.After(10 * time.Second)
	log.Println("done")
	return "done", nil
}

func NewSleepyFn() fn.Fn {
	return &sleepyFn{}
}

type echoFn struct{}

func (*echoFn) Invoke(ctx context.Context, input interface{}) (interface{}, error) {
	return fmt.Sprintf("echo: %s", input), nil
}

func NewEchoFn() fn.Fn {
	return &echoFn{}
}
