package runner_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/fnrun/fnrun/pkg/fn"
	"github.com/fnrun/fnrun/pkg/run"
	"github.com/fnrun/fnrun/pkg/run/config"
	"github.com/fnrun/fnrun/pkg/run/fn/loader"
	"github.com/fnrun/fnrun/pkg/run/middleware/pipeline"
	"github.com/fnrun/fnrun/pkg/run/runner"
	"github.com/fnrun/fnrun/pkg/run/source/cron"
	sourceloader "github.com/fnrun/fnrun/pkg/run/source/loader"
)

func newRegistry() run.Registry {
	reg := run.NewRegistry()
	reg.RegisterSourceWithRegistry("source", sourceloader.New)
	reg.RegisterSource("cron", cron.New)
	reg.RegisterFnWithRegistry("fn", loader.New)
	reg.RegisterFn("prefix", newPrefixFn)
	reg.RegisterMiddlewareWithRegistry("middleware", pipeline.NewWithRegistry)
	reg.RegisterMiddleware("wrap", newWrapMiddleware)

	return reg
}

func TestConfigure_invalidConfiguration(t *testing.T) {
	r := runner.New(newRegistry())

	err := config.Configure(r, nil)
	if err == nil {
		t.Error("expected config.Configure to return an error but it did not")
	}
}

func TestConfigureMap_noSourceConfiguration(t *testing.T) {
	r := runner.New(newRegistry())

	err := config.Configure(r, map[string]interface{}{
		"fn": "prefix",
	})

	if err == nil {
		t.Error("expected config.Configure to return an error but it did not")
	}

	got := err.Error()
	want := "source is a required configuration key"

	if got != want {
		t.Errorf("unexpected error message: want %q, got %q", want, got)
	}
}

func TestConfigureMap_noFnConfiguration(t *testing.T) {
	r := runner.New(newRegistry())

	err := config.Configure(r, map[string]interface{}{
		"source": map[string]interface{}{
			"cron": "@every 1s",
		},
	})

	if err == nil {
		t.Error("expected config.Configure to return an error but it did not")
	}

	got := err.Error()
	want := "fn is a required configuration key"

	if got != want {
		t.Errorf("unexpected error message: want %q, got %q", want, got)
	}
}

func TestConfigureMap_missingFn(t *testing.T) {
	reg := run.NewRegistry()
	// NOTE: no "fn" applied to the registry (the fn loader)
	reg.RegisterSourceWithRegistry("source", sourceloader.New)
	reg.RegisterSource("cron", cron.New)
	reg.RegisterFn("prefix", newPrefixFn)
	r := runner.New(reg)

	err := config.Configure(r, map[string]interface{}{
		"source": map[string]interface{}{
			"cron": "@every 1s",
		},
		"fn": "other-fn",
	})

	if err == nil {
		t.Error("expected config.Configure to return an error but it did not")
	}

	got := err.Error()
	want := `a registered fn not found for key "fn"`

	if got != want {
		t.Errorf("unexpected error message: want %q, got %q", want, got)
	}
}

func TestConfigureMap_invalidConfigForFn(t *testing.T) {
	r := runner.New(newRegistry())

	err := config.Configure(r, map[string]interface{}{
		"source": map[string]interface{}{
			"cron": "@every 1s",
		},
		"fn": "prefix",
	})

	if err == nil {
		t.Error("expected config.Configure to return an error but it did not")
	}

	got := err.Error()
	want := "*runner_test.prefixFn could not be configured with object of type <nil>"

	if got != want {
		t.Errorf("unexpected error message: want %q, got %q", want, got)
	}
}

func TestConfigureMap_missingMiddleware(t *testing.T) {
	reg := run.NewRegistry()
	reg.RegisterSourceWithRegistry("source", sourceloader.New)
	reg.RegisterSource("cron", cron.New)
	reg.RegisterFnWithRegistry("fn", loader.New)
	reg.RegisterFn("prefix", newPrefixFn)
	// NOTE: no middleware loader
	r := runner.New(reg)

	err := config.Configure(r, map[string]interface{}{
		"source": map[string]interface{}{
			"cron": "@every 1s",
		},
		"fn": map[string]interface{}{
			"prefix": "fn-prefix",
		},
		"middleware": []interface{}{
			map[string]interface{}{"wrap": "nameA"},
		},
	})

	if err == nil {
		t.Error("expected config.Configure to return an error but it did not")
	}

	got := err.Error()
	want := `a registered middleware not found for key "middleware"`

	if got != want {
		t.Errorf("unexpected error message: want %q, got %q", want, got)
	}
}

func TestConfigureMap_invalidConfigForMiddleware(t *testing.T) {
	r := runner.New(newRegistry())

	err := config.Configure(r, map[string]interface{}{
		"source": map[string]interface{}{
			"cron": "@every 1s",
		},
		"fn": map[string]interface{}{
			"prefix": "fn-prefix",
		},
		"middleware": []interface{}{
			map[string]interface{}{"wrap": true},
		},
	})

	if err == nil {
		t.Fatal("expected config.Configure to return an error but it did not")
	}

	got := err.Error()
	want := "*runner_test.wrapMiddleware could not be configured with object of type bool"

	if got != want {
		t.Errorf("unexpected error message: want %q, got %q", want, got)
	}
}

func TestConfigureMap_missingSource(t *testing.T) {
	reg := run.NewRegistry()
	reg.RegisterSource("cron", cron.New)
	reg.RegisterFnWithRegistry("fn", loader.New)
	reg.RegisterFn("prefix", newPrefixFn)
	reg.RegisterMiddlewareWithRegistry("middleware", pipeline.NewWithRegistry)
	reg.RegisterMiddleware("wrap", newWrapMiddleware)

	r := runner.New(reg)

	err := config.Configure(r, map[string]interface{}{
		"source": map[string]interface{}{
			"cron": "@every 1s",
		},
		"fn": map[string]interface{}{
			"prefix": "fn-prefix",
		},
		"middleware": []interface{}{
			map[string]interface{}{"wrap": "nameA"},
		},
	})

	if err == nil {
		t.Error("expected config.Configure to return an error but it did not")
	}

	got := err.Error()
	want := `a registered source not found for key "source"`

	if got != want {
		t.Errorf("unexpected error message: want %q, got %q", want, got)
	}
}

func TestConfigureMap_invalidConfigForSource(t *testing.T) {
	r := runner.New(newRegistry())

	err := config.Configure(r, map[string]interface{}{
		"source": map[string]interface{}{
			"cron": true,
		},
		"fn": map[string]interface{}{
			"prefix": "fn-prefix",
		},
		"middleware": []interface{}{
			map[string]interface{}{"wrap": "nameA"},
		},
	})

	if err == nil {
		t.Fatal("expected config.Configure to return an error but it did not")
	}

	got := err.Error()
	want := "*cron.cronSource could not be configured with object of type bool"

	if got != want {
		t.Errorf("unexpected error message: want %q, got %q", want, got)
	}
}

func TestRun_withoutMiddleware(t *testing.T) {
	s := newChanSource()
	reg := newRegistry()
	reg.RegisterSource("chan", func() run.Source { return s })

	r := runner.New(reg)

	err := config.Configure(r, map[string]interface{}{
		"source": "chan",
		"fn": map[string]interface{}{
			"prefix": "fn-prefix",
		},
	})

	if err != nil {
		t.Fatalf("config.Configure returned error: %+v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	go func() {
		r.Run(ctx)
	}()

	s.InputCh <- "some input"

	select {
	case <-ctx.Done():
		t.Fatal("output not received in timeout period")

	case result := <-s.OutputCh:
		got := result
		want := "fn-prefix: some input"
		if got != want {
			t.Errorf("unexpected output: want %q, got %q", want, got)
		}
	}
}

func TestRun(t *testing.T) {
	s := newChanSource()
	reg := newRegistry()
	reg.RegisterSource("chan", func() run.Source { return s })

	r := runner.New(reg)

	err := config.Configure(r, map[string]interface{}{
		"source": "chan",
		"fn": map[string]interface{}{
			"prefix": "fn-prefix",
		},
		"middleware": []interface{}{
			map[string]interface{}{
				"wrap": "NAME",
			},
		},
	})

	if err != nil {
		t.Fatalf("config.Configure returned error: %+v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	go func() {
		r.Run(ctx)
	}()

	s.InputCh <- "some input"

	select {
	case <-ctx.Done():
		t.Fatal("output not received in timeout period")

	case result := <-s.OutputCh:
		got := result
		want := "NAME fn-prefix: NAME some input"
		if got != want {
			t.Errorf("unexpected output: want %q, got %q", want, got)
		}
	}
}

// -----------------------------------------------------------------------------
// Test components

type prefixFn struct {
	prefix string
}

func (p *prefixFn) ConfigureString(prefix string) error {
	p.prefix = prefix
	return nil
}

func (p *prefixFn) RequiresConfig() bool {
	return true
}

func (p *prefixFn) Invoke(ctx context.Context, input interface{}) (interface{}, error) {
	message := fmt.Sprintf("%s: %v", p.prefix, input)
	return message, nil
}

func newPrefixFn() fn.Fn {
	return &prefixFn{}
}

type wrapMiddleware struct {
	name string
}

func (w *wrapMiddleware) ConfigureString(name string) error {
	w.name = name
	return nil
}

func (w *wrapMiddleware) RequiresConfig() bool {
	return true
}

func (w *wrapMiddleware) Invoke(ctx context.Context, input interface{}, f fn.Fn) (interface{}, error) {
	newInput := fmt.Sprintf("%s %v", w.name, input)
	output, err := f.Invoke(ctx, newInput)
	if err != nil {
		return output, err
	}
	return fmt.Sprintf("%s %v", w.name, output), nil
}

func newWrapMiddleware() run.Middleware {
	return &wrapMiddleware{}
}

type chanSource struct {
	InputCh  chan interface{}
	OutputCh chan interface{}
}

func (c *chanSource) Serve(ctx context.Context, f fn.Fn) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case input := <-c.InputCh:
			output, err := f.Invoke(ctx, input)
			if err != nil {
				continue
			}
			c.OutputCh <- output
		}
	}
}

func newChanSource() *chanSource {
	return &chanSource{
		InputCh:  make(chan interface{}, 1),
		OutputCh: make(chan interface{}, 1),
	}
}
