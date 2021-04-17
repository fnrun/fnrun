// Package middleware provides a function that is wrapped by a middleware.
package middleware

import (
	"context"

	"github.com/fnrun/fnrun/fn"
	"github.com/fnrun/fnrun/run"
)

type middlewareFn struct {
	middleware run.Middleware
	fn         fn.Fn
}

func (m *middlewareFn) Invoke(ctx context.Context, input interface{}) (interface{}, error) {
	return m.middleware.Invoke(ctx, input, m.fn)
}

// New creates an Fn that wraps fn with middleware.
func New(middleware run.Middleware, fn fn.Fn) fn.Fn {
	return &middlewareFn{
		middleware: middleware,
		fn:         fn,
	}
}
