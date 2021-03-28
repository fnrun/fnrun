package loader_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/fnrun/fnrun/pkg/fn"
	"github.com/fnrun/fnrun/pkg/run"
	"github.com/fnrun/fnrun/pkg/run/config"
	"github.com/fnrun/fnrun/pkg/run/fn/identity"
	"github.com/fnrun/fnrun/pkg/run/source/loader"
)

func TestConfigure_withInvalidConfig(t *testing.T) {
	reg := run.NewRegistry()
	src := loader.New(reg)
	err := config.Configure(src, nil)

	if err == nil {
		t.Error("expected config.Configure to return an error but it did not")
	}
}

func TestConfigureString_missingSourceKey(t *testing.T) {
	reg := run.NewRegistry()
	src := loader.New(reg)
	err := config.Configure(src, "some-source")

	if err == nil {
		t.Fatal("expected config.Configure to return an error but it did not")
	}

	got := err.Error()
	want := `a registered source not found for key "some-source"`

	if got != want {
		t.Errorf("unexpected error message: want %q, got %q", want, got)
	}
}

func TestConfigureString_invalidSourceConfig(t *testing.T) {
	reg := run.NewRegistry()
	reg.RegisterSource("configurable", newConfigurableSource)
	src := loader.New(reg)
	err := config.Configure(src, "configurable")

	if err == nil {
		t.Fatal("expected config.Configure to return an error but it did not")
	}

	got := err.Error()
	want := `*loader_test.configurableSource could not be configured with object of type <nil>`

	if got != want {
		t.Errorf("unexpected error message: want %q, got %q", want, got)
	}
}

func TestConfigureMap_multipleKeys(t *testing.T) {
	reg := run.NewRegistry()
	reg.RegisterSource("configurable", newConfigurableSource)
	src := loader.New(reg)

	err := config.Configure(src, map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
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

func TestServe_configuredByString(t *testing.T) {
	reg := run.NewRegistry()
	c := newChanSource()
	reg.RegisterSource("chan", func() run.Source { return c })

	src := loader.New(reg)
	err := config.Configure(src, "chan")
	if err != nil {
		t.Fatalf("config.Configure returned error: %+v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	go func() {
		src.Serve(ctx, identity.New())
	}()

	c.InputCh <- "some input"

	select {
	case <-ctx.Done():
		t.Fatal("execution was not completed within the timeout period")
	case got := <-c.OutputCh:
		want := "invoked by Default: some input"
		if got != want {
			t.Errorf("unexpected result: want %q, got %q", want, got)
		}
	}
}

func TestServe_configuredByMap(t *testing.T) {
	reg := run.NewRegistry()
	c := newChanSource()
	reg.RegisterSource("chan", func() run.Source { return c })

	src := loader.New(reg)
	err := config.Configure(src, map[string]interface{}{
		"chan": "Custom",
	})
	if err != nil {
		t.Fatalf("config.Configure returned error: %+v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	go func() {
		src.Serve(ctx, identity.New())
	}()

	c.InputCh <- "some input"

	select {
	case <-ctx.Done():
		t.Fatal("execution was not completed within the timeout period")
	case got := <-c.OutputCh:
		want := "invoked by Custom: some input"
		if got != want {
			t.Errorf("unexpected result: want %q, got %q", want, got)
		}
	}
}

// -----------------------------------------------------------------------------
// Test components

type configurableSource struct {
	Name string
}

func (c *configurableSource) ConfigureString(name string) error {
	c.Name = name
	return nil
}

func (c *configurableSource) RequiresConfig() bool {
	return true
}

func (c *configurableSource) Serve(context.Context, fn.Fn) error {
	return errors.New("not implemented")
}

func newConfigurableSource() run.Source {
	return &configurableSource{}
}

type chanSource struct {
	InputCh  chan interface{}
	OutputCh chan interface{}
	Name     string
}

func (c *chanSource) ConfigureString(name string) error {
	c.Name = name
	return nil
}

func (c *chanSource) Serve(ctx context.Context, f fn.Fn) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case input := <-c.InputCh:
			output, err := f.Invoke(ctx, fmt.Sprintf("invoked by %s: %v", c.Name, input))
			if err != nil {
				continue
			}
			c.OutputCh <- output
		}
	}
}

func newChanSource() *chanSource {
	return &chanSource{
		Name:     "Default",
		InputCh:  make(chan interface{}, 1),
		OutputCh: make(chan interface{}, 1),
	}
}
