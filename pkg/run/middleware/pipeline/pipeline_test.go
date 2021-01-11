package pipeline

import (
	"context"
	"strings"
	"testing"

	"github.com/fnrun/fnrun/pkg/fn"
	"github.com/fnrun/fnrun/pkg/run"
)

func newRegistry() run.Registry {
	r := run.NewRegistry()
	r.RegisterMiddleware("sandwich", NewSandwichMiddleware)
	r.RegisterMiddleware("upper", NewUpperMiddleware)
	r.RegisterMiddleware("prepend", NewPrependMiddleware)
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

func NewPrependMiddleware() run.Middleware {
	return &prependMiddleware{}
}
