package run

import (
	"context"
	"testing"

	"github.com/fnrun/fnrun/fn"
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
	r.RegisterFn("test", NewTestFn)

	_, exists := r.FindFn("test")
	if !exists {
		t.Error("expected key not found in registry")
	}

	_, exists = r.FindFn("unknown")
	if exists {
		t.Error("registry unexpectedly returned a value")
	}
}

func TestRegistry_RegisterSourceWithRegistry(t *testing.T) {
	r := NewRegistry()
	r.RegisterSourceWithRegistry("sourceWithRegistry", NewTestSourceWithRegistry)

	factory, exists := r.FindSource("sourceWithRegistry")
	if !exists {
		t.Error("expected key not found in registry")
	}

	source := factory()
	testSource, ok := source.(*testSource)
	if !ok {
		t.Errorf("expected source to be a testSource but was a %T", testSource)
	}

	want := r
	got := testSource.Registry

	if got != want {
		t.Errorf("testSource did not have expected registry: want %v, got %v", want, got)
	}
}

func TestRegistry_RegisterMiddlewareWithRegistry(t *testing.T) {
	r := NewRegistry()
	r.RegisterMiddlewareWithRegistry("middlewareWithRegistry", NewTestMiddlewareWithRegistry)

	factory, exists := r.FindMiddleware("middlewareWithRegistry")
	if !exists {
		t.Error("expected key not found in registry")
	}

	middleware := factory()
	testMiddleware, ok := middleware.(*testMiddleware)
	if !ok {
		t.Errorf("expected source to be a testSource but was a %T", testMiddleware)
	}

	want := r
	got := testMiddleware.Registry

	if got != want {
		t.Errorf("testSource did not have expected registry: want %v, got %v", want, got)
	}
}

func TestRegistry_RegisterFnWithRegistry(t *testing.T) {
	r := NewRegistry()
	r.RegisterFnWithRegistry("fnWithRegistry", NewTestFnWithRegistry)

	factory, exists := r.FindFn("fnWithRegistry")
	if !exists {
		t.Error("expected key not found in registry")
	}

	f := factory()
	testFn, ok := f.(*testFn)
	if !ok {
		t.Errorf("expected source to be a testSource but was a %T", testFn)
	}

	want := r
	got := testFn.Registry

	if got != want {
		t.Errorf("testSource did not have expected registry: want %v, got %v", want, got)
	}
}

// -----------------------------------------------------------------------------
// Test types

type testSource struct {
	Registry Registry
}

func (t *testSource) Serve(context.Context, fn.Fn) error {
	return nil
}

var _ Source = (*testSource)(nil)

func NewTestSource() Source {
	return &testSource{}
}

func NewTestSourceWithRegistry(registry Registry) Source {
	return &testSource{
		Registry: registry,
	}
}

// -------------------------------------
// testMiddleware

type testMiddleware struct {
	Registry Registry
}

func (t *testMiddleware) Invoke(ctx context.Context, input interface{}, f fn.Fn) (interface{}, error) {
	return input, nil
}

var _ Middleware = (*testMiddleware)(nil)

func NewTestMiddleware() Middleware {
	return &testMiddleware{}
}

func NewTestMiddlewareWithRegistry(registry Registry) Middleware {
	return &testMiddleware{
		Registry: registry,
	}
}

// -------------------------------------
// testFn

type testFn struct {
	Registry Registry
}

func (*testFn) Invoke(ctx context.Context, input interface{}) (interface{}, error) {
	return input, nil
}

var _ fn.Fn = (*testFn)(nil)

func NewTestFn() fn.Fn {
	return &testFn{}
}

func NewTestFnWithRegistry(registry Registry) fn.Fn {
	return &testFn{
		Registry: registry,
	}
}
