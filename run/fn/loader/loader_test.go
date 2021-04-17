package loader_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/fnrun/fnrun/fn"
	"github.com/fnrun/fnrun/run"
	"github.com/fnrun/fnrun/run/config"
	"github.com/fnrun/fnrun/run/fn/loader"
)

func TestConfigure_invalidValue(t *testing.T) {
	r := run.NewRegistry()
	f := loader.New(r)
	err := config.Configure(f, nil)

	if err == nil {
		t.Error("expected config.Configure to return a value but it did not")
	}
}

func TestConfigureString_missingFn(t *testing.T) {
	r := run.NewRegistry()
	f := loader.New(r)
	err := config.Configure(f, "missing-fn")

	if err == nil {
		t.Fatal("expected config.Configure to return an error but it did not")
	}

	got := err.Error()
	want := `a registered fn not found for key "missing-fn"`
	if got != want {
		t.Errorf("unexpected error message: want %q, got %q", want, got)
	}
}

func TestConfigureString_invalidFnConfiguration(t *testing.T) {
	r := run.NewRegistry()
	r.RegisterFn("prefix", newPrefixFn)
	f := loader.New(r)

	err := config.Configure(f, "prefix")

	if err == nil {
		t.Fatal("expected config.Configure to return an error but it did not")
	}

	got := err.Error()
	want := "*loader_test.prefixFn could not be configured with object of type <nil>"
	if got != want {
		t.Errorf("unexpected error message: want %q, got %q", want, got)
	}
}

func TestConfigureMap_multipleKeys(t *testing.T) {
	r := run.NewRegistry()
	f := loader.New(r)

	err := config.Configure(f, map[string]interface{}{
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

func TestConfigureMap_missingFn(t *testing.T) {
	r := run.NewRegistry()
	f := loader.New(r)

	err := config.Configure(f, map[string]interface{}{
		"prefix": "some-prefix",
	})
	if err == nil {
		t.Fatal("expected config.Configure to return an error but it did not")
	}

	got := err.Error()
	want := `a registered fn not found for key "prefix"`
	if got != want {
		t.Errorf("unexpected error message: want %q, got %q", want, got)
	}
}

func TestConfigureMap_invalidFnConfiguration(t *testing.T) {
	r := run.NewRegistry()
	r.RegisterFn("prefix", newPrefixFn)
	f := loader.New(r)

	err := config.Configure(f, map[string]interface{}{
		"prefix": true,
	})
	if err == nil {
		t.Fatal("expected config.Configure to return an error but it did not")
	}

	want := "could not be configured"
	if !strings.Contains(err.Error(), want) {
		t.Errorf("unexpected error message: wanted to contain %q, got %q", want, err.Error())
	}
}

func TestInvoke_configuredViaString(t *testing.T) {
	r := run.NewRegistry()
	r.RegisterFn("echo", newEchoFn)
	f := loader.New(r)

	err := config.Configure(f, "echo")
	if err != nil {
		t.Fatalf("config.Configure returned error: %+v", err)
	}

	output, err := f.Invoke(context.Background(), "some input")
	if err != nil {
		t.Errorf("Invoke returned error: %+v", err)
	}

	want := "some input"
	got := output

	if got != want {
		t.Errorf("unexpected output: want %q, got %q", want, got)
	}
}

func TestInvoke_configuredViaMap(t *testing.T) {
	r := run.NewRegistry()
	r.RegisterFn("prefix", newPrefixFn)
	f := loader.New(r)

	err := config.Configure(f, map[string]interface{}{
		"prefix": "my-prefix",
	})
	if err != nil {
		t.Fatalf("config.Configure returned error: %+v", err)
	}

	output, err := f.Invoke(context.Background(), "some input")
	if err != nil {
		t.Errorf("Invoke returned error: %+v", err)
	}

	want := "my-prefix: some input"
	got := output

	if got != want {
		t.Errorf("unexpected output: want %q, got %q", want, got)
	}
}

// -----------------------------------------------------------------------------
// Test functions

type prefixFn struct {
	prefix string
}

func (p *prefixFn) Invoke(ctx context.Context, input interface{}) (interface{}, error) {
	message := fmt.Sprintf("%s: %v", p.prefix, input)
	return message, nil
}

func (p *prefixFn) ConfigureString(prefix string) error {
	p.prefix = prefix
	return nil
}

func (p *prefixFn) RequiresConfig() bool {
	return true
}

func newPrefixFn() fn.Fn {
	return &prefixFn{}
}

type echoFn struct{}

func (*echoFn) Invoke(ctx context.Context, input interface{}) (interface{}, error) {
	return input, nil
}
func newEchoFn() fn.Fn {
	return &echoFn{}
}
