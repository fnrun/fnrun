package middleware

import (
	"context"
	"testing"

	"github.com/fnrun/fnrun/fn"
	"github.com/fnrun/fnrun/run"
)

func TestNew(t *testing.T) {
	testMiddleware := NewTestMiddleware()
	testFn := NewTestFn()

	f := New(testMiddleware, testFn)

	output, err := f.Invoke(context.Background(), "input")
	if err != nil {
		t.Error(err)
	}

	got := output.(string)
	want := "before input:result after"

	if want != got {
		t.Errorf("want %q, got %q", want, got)
	}
}

// -----------------------------------------------------------------------------
// Test types

type testMiddleware struct{}

func (t *testMiddleware) Invoke(ctx context.Context, input interface{}, f fn.Fn) (interface{}, error) {
	s := input.(string)
	output, err := f.Invoke(ctx, "before "+s)
	s = output.(string)
	return s + " after", err
}

func NewTestMiddleware() run.Middleware {
	return &testMiddleware{}
}

type testFn struct{}

func (*testFn) Invoke(ctx context.Context, input interface{}) (interface{}, error) {
	s := input.(string)
	return s + ":result", nil
}

func NewTestFn() fn.Fn {
	return &testFn{}
}
