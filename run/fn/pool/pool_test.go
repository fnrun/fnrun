package pool_test

import (
	"context"
	"fmt"
	"log"
	"sync"
	"testing"
	"time"

	"github.com/fnrun/fnrun/fn"
	"github.com/fnrun/fnrun/run"
	"github.com/fnrun/fnrun/run/config"
	"github.com/fnrun/fnrun/run/fn/pool"
	"github.com/mitchellh/mapstructure"
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
	wg := &sync.WaitGroup{}
	wg.Add(2)

	go func() {
		wg.Done()
		p.Invoke(ctx, "first input")
	}()

	go func() {
		wg.Done()
		p.Invoke(ctx, "second input")
	}()

	// The following is cheating a little, but it is to get around a race
	// race condition with the kicking off of the go routines above.
	wg.Wait()
	<-time.After(1 * time.Millisecond)

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

func TestConfigure_invalidConfig(t *testing.T) {
	r := run.NewRegistry()
	p := pool.New(r)

	err := config.Configure(p, nil)
	if err == nil {
		t.Error("expected config.Configure to return an error but it did not")
	}
}

func TestConfigure_unknownFn(t *testing.T) {
	r := run.NewRegistry()
	p := pool.New(r)

	err := config.Configure(p, map[string]interface{}{
		"template": "unknown-fn",
	})
	if err == nil {
		t.Fatal("expected config.Configure to return an error but it did not")
	}

	got := err.Error()
	want := `a registered fn not found for key "unknown-fn"`

	if got != want {
		t.Errorf("unexecpted error message: want %q, got %q", want, got)
	}
}

func TestConfigureMap_configWithInvalidUndecodableValue(t *testing.T) {
	r := run.NewRegistry()
	p := pool.New(r)

	err := config.Configure(p, map[string]interface{}{
		"concurrency": "not an int",
	})
	if err == nil {
		t.Fatal("expected config.Configure to return an error and it did not")
	}

	got := err.Error()
	want := "1 error(s) decoding:\n\n* 'concurrency' expected type 'int', got unconvertible type 'string', value: 'not an int'"

	if got != want {
		t.Errorf("unexpected error message: want %q, got %q", want, got)
	}
}

func TestConfigureMap_configWithInvalidDurationString(t *testing.T) {
	r := run.NewRegistry()
	p := pool.New(r)

	err := config.Configure(p, map[string]interface{}{
		"maxWaitDuration": "an invalid value",
	})
	if err == nil {
		t.Fatal("expected config.Configure to return an error and it did not")
	}

	got := err.Error()
	want := `time: invalid duration "an invalid value"`

	if got != want {
		t.Errorf("unexpected error message: want %q, got %q", want, got)
	}
}

func TestConfigureMap_templateMapConfig(t *testing.T) {
	r := run.NewRegistry()
	r.RegisterFn("map", func() fn.Fn { return &mapFn{} })
	p := pool.New(r)

	err := config.Configure(p, map[string]interface{}{
		"template": map[string]interface{}{
			"map": map[string]interface{}{
				"count": 3,
				"name":  "some name",
			},
		},
	})
	if err != nil {
		t.Fatalf("config.Configure returned error: %+v", err)
	}

	output, err := p.Invoke(context.Background(), "some input")
	if err != nil {
		t.Errorf("Invoke returned error: %+v", err)
	}

	want := "count: 3, name: some name"
	got, ok := output.(string)
	if !ok {
		t.Fatalf("expected output to be a string but it was a %T", output)
	}

	if got != want {
		t.Errorf("unexpected output: want %q, got %q", want, got)
	}
}

func TestConfigureMap_templateMapConfig_multipleValues(t *testing.T) {
	r := run.NewRegistry()
	p := pool.New(r)

	err := config.Configure(p, map[string]interface{}{
		"template": map[string]interface{}{
			"key1": "some value",
			"key2": "other value",
		},
	})

	if err == nil {
		t.Fatal("expected config.Configure to return an error but it did not")
	}

	got := err.Error()
	want := "expected map to have exactly one entry"

	if got != want {
		t.Errorf("unexpected error message: want %q, got %q", want, got)
	}
}

func TestConfigureMap_templateInvalidType(t *testing.T) {
	r := run.NewRegistry()
	p := pool.New(r)

	err := config.Configure(p, map[string]interface{}{
		"template": 3,
	})

	if err == nil {
		t.Fatal("expected config.Configure to return an error but it did not")
	}

	got := err.Error()
	want := "unsupported config type"

	if got != want {
		t.Errorf("unexpected error message: want %q, got %q", want, got)
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

type mapFn struct {
	Count int    `mapstructure:"count"`
	Name  string `mapstructure:"name"`
}

func (m *mapFn) ConfigureMap(configMap map[string]interface{}) error {
	return mapstructure.Decode(configMap, m)
}

func (m *mapFn) Invoke(context.Context, interface{}) (interface{}, error) {
	return fmt.Sprintf("count: %d, name: %s", m.Count, m.Name), nil
}
