package run

import (
	"context"
	"testing"

	"github.com/fnrun/fnrun/pkg/fn"
)

func TestRegistry_FindSource(t *testing.T) {
	r := NewRegistry()
	r.RegisterSource("test", NewTestSource)

	_, exists := r.FindSource("test")
	if !exists {
		t.Error("expected key not found in registry")
	}

	_, exists = r.FindSource("unknown")
	if exists {
		t.Error("registry unexpectedly returned a value")
	}
}

func TestRegistry_FindMiddleware(t *testing.T) {
	r := NewRegistry()
	r.RegisterMiddleware("test", NewTestMiddleware)

	_, exists := r.FindMiddleware("test")
	if !exists {
		t.Error("expected key not found in registry")
	}

	_, exists = r.FindMiddleware("unknown")
	if exists {
		t.Error("registry unexpectedly returned a value")
	}
}

func TestRegistry_FindFn(t *testing.T) {
	r := NewRegistry()
	r.RegisterFn("test", newTestFn)

	_, exists := r.FindFn("test")
	if !exists {
		t.Error("expected key not found in registry")
	}

	_, exists = r.FindFn("unknown")
	if exists {
		t.Error("registry unexpectedly returned a value")
	}
}

// -----------------------------------------------------------------------------
// Test types

type testSource struct{}

func (t *testSource) Serve(context.Context, fn.Fn) error {
	return nil
}

var _ Source = (*testSource)(nil)

func NewTestSource() Source {
	return &testSource{}
}

// -------------------------------------
// testMiddleware

type testMiddleware struct{}

func (t *testMiddleware) Invoke(ctx context.Context, input interface{}, f fn.Fn) (interface{}, error) {
	return input, nil
}

var _ Middleware = (*testMiddleware)(nil)

func NewTestMiddleware() Middleware {
	return &testMiddleware{}
}

// -------------------------------------
// testFn

type testFn struct{}

func (*testFn) Invoke(ctx context.Context, input interface{}) (interface{}, error) {
	return input, nil
}

var _ fn.Fn = (*testFn)(nil)

func newTestFn() fn.Fn {
	return &testFn{}
}
