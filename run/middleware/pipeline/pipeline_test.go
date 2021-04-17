package pipeline

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/fnrun/fnrun/fn"
	"github.com/fnrun/fnrun/run"
	"github.com/fnrun/fnrun/run/config"
	"github.com/mitchellh/mapstructure"
)

func newRegistry() run.Registry {
	r := run.NewRegistry()
	r.RegisterMiddleware("sandwich", NewSandwichMiddleware)
	r.RegisterMiddleware("upper", NewUpperMiddleware)
	r.RegisterMiddleware("prepend", NewPrependMiddleware)
	r.RegisterMiddleware("map", NewMapConfigMiddleware)
	return r
}

func TestInvoke_unconfigured(t *testing.T) {
	r := run.NewRegistry()
	m := NewWithRegistry(r)
	f := &echoFn{}

	output, err := m.Invoke(context.Background(), "some input", f)
	if err != nil {
		t.Fatalf("Invoke returned error: %#v", err)
	}

	want := "some input"
	got := output.(string)

	if got != want {
		t.Errorf("output of Invoke: want %q; got %q", want, got)
	}
}

func TestConfigureArray_missingMiddleware(t *testing.T) {
	r := run.NewRegistry()
	m := NewWithRegistry(r)

	err := config.Configure(m, []interface{}{"some-middleware"})
	if err == nil {
		t.Error("expected config.Configure to return an error but it did not")
	}
}

func TestConfigureArray_invalidMiddlewareStringConfig(t *testing.T) {
	r := newRegistry()
	m := NewWithRegistry(r)

	err := config.Configure(m, []interface{}{"prepend"})
	if err == nil {
		t.Error("expected config.Configure to return an error but it did not")
	}
}

func TestConfigureArray_middlewareMapConfig(t *testing.T) {
	r := newRegistry()
	m := NewWithRegistry(r)

	cfg := map[string]interface{}{
		"count": 3,
		"name":  "some value",
	}

	err := config.Configure(m, []interface{}{
		map[string]interface{}{
			"map": cfg,
		},
	})

	if err != nil {
		t.Fatalf("config.Configure returned error: %+v", err)
	}

	output, err := m.Invoke(context.Background(), nil, &echoFn{})
	if err != nil {
		t.Errorf("Invoke returned error: %+v", err)
	}

	got, ok := output.(string)
	if !ok {
		t.Errorf("Expected output to be a string but it was a %T", output)
	}
	want := "count: 3 ; name: some value"
	if got != want {
		t.Errorf("want: %q; got %q", want, got)
	}
}

func TestConfigureArray_mapWithMultiKeysAsSingleMiddlewareConfig(t *testing.T) {
	r := newRegistry()
	m := NewWithRegistry(r)

	cfg := map[string]interface{}{
		"count": 3,
		"name":  "some value",
	}

	got := config.Configure(m, []interface{}{
		map[string]interface{}{
			"map":       cfg,
			"something": "else",
		},
	})

	want := errSingleKey

	if got != want {
		t.Errorf("want %+v; got %+v", want, got)
	}
}

func TestConfigureArray_mapWithMissingMiddleware(t *testing.T) {
	r := newRegistry()
	m := NewWithRegistry(r)

	err := config.Configure(m, []interface{}{
		map[string]interface{}{
			"unknown": map[string]interface{}{"some": "value"},
		},
	})

	if err == nil {
		t.Fatalf("Expected config.Configure to return an error but it did not")
	}

	got := err.Error()
	want := "no middleware registered with key unknown"

	if got != want {
		t.Errorf("error message: want %q, got %q", want, got)
	}
}

func TestConfigureArray_mapWithInvalidMiddlewareConfig(t *testing.T) {
	r := newRegistry()
	m := NewWithRegistry(r)

	err := config.Configure(m, []interface{}{
		map[string]interface{}{
			"map": 3,
		},
	})

	if err == nil {
		t.Error("expected config.Configure to return an error but it did not")
	}
}

func TestConfigureArray_invalidMiddlewareType(t *testing.T) {
	r := newRegistry()
	m := NewWithRegistry(r)

	err := config.Configure(m, []interface{}{
		3,
	})

	if err == nil {
		t.Fatal("expected config.Configure to return an error but it did not")
	}

	got := err.Error()
	want := "wrong middleware configuration type: int, expected string or object"
	if got != want {
		t.Errorf("error message: want %q; got %q", want, got)
	}
}

func TestInvoke(t *testing.T) {
	r := newRegistry()

	t.Run("upper before sandwich", func(t *testing.T) {
		m := NewWithRegistry(r)
		p := m.(*pipelineMiddleware)
		p.ConfigureArray([]interface{}{"upper", "sandwich"})
		f := &echoFn{}

		output, err := p.Invoke(context.Background(), "hello", f)
		if err != nil {
			t.Errorf("Invoke returned error: %#v", err)
		}

		want := "before: HELLO :after"
		got := output.(string)

		if got != want {
			t.Errorf("output from Invoke: want %q, got %q", want, got)
		}
	})

	t.Run("sandwich before upper", func(t *testing.T) {
		m := NewWithRegistry(r)
		p := m.(*pipelineMiddleware)
		p.ConfigureArray([]interface{}{
			map[string]interface{}{"prepend": "first "},
			"sandwich",
			"upper",
		})
		f := &echoFn{}

		output, err := p.Invoke(context.Background(), "hello", f)
		if err != nil {
			t.Errorf("Invoke returned error: %#v", err)
		}

		want := "BEFORE: FIRST HELLO :after"
		got := output.(string)

		if got != want {
			t.Errorf("output from Invoke: want %q, got %q", want, got)
		}
	})
}

// -----------------------------------------------------------------------------
// Sample

type echoFn struct{}

func (*echoFn) Invoke(ctx context.Context, input interface{}) (interface{}, error) {
	return input, nil
}

type sandwichMiddleware struct{}

func (*sandwichMiddleware) Invoke(ctx context.Context, input interface{}, f fn.Fn) (interface{}, error) {
	inputStr := input.(string)

	output, err := f.Invoke(ctx, "before: "+inputStr)

	outputStr := output.(string)

	return outputStr + " :after", err
}

func NewSandwichMiddleware() run.Middleware {
	return &sandwichMiddleware{}
}

type upperMiddleware struct{}

func (*upperMiddleware) Invoke(ctx context.Context, input interface{}, f fn.Fn) (interface{}, error) {
	inputStr := input.(string)
	return f.Invoke(ctx, strings.ToUpper(inputStr))
}

func NewUpperMiddleware() run.Middleware {
	return &upperMiddleware{}
}

type prependMiddleware struct {
	prefix string
}

func (p *prependMiddleware) ConfigureString(prefix string) error {
	p.prefix = prefix
	return nil
}

func (p *prependMiddleware) Invoke(ctx context.Context, input interface{}, f fn.Fn) (interface{}, error) {
	return f.Invoke(ctx, p.prefix+input.(string))
}

func (p *prependMiddleware) RequiresConfig() bool {
	return true
}

func NewPrependMiddleware() run.Middleware {
	return &prependMiddleware{}
}

type mapConfigMiddleware struct {
	Count int    `mapstructure:"count"`
	Name  string `mapstructure:"name"`
}

func (m *mapConfigMiddleware) RequiresConfig() bool {
	return true
}

func (m *mapConfigMiddleware) ConfigureMap(configMap map[string]interface{}) error {
	return mapstructure.Decode(configMap, m)
}

func (m *mapConfigMiddleware) Invoke(context.Context, interface{}, fn.Fn) (interface{}, error) {
	message := fmt.Sprintf("count: %d ; name: %s", m.Count, m.Name)
	return message, nil
}

func NewMapConfigMiddleware() run.Middleware {
	return &mapConfigMiddleware{}
}
