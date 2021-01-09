package run

import (
	"context"

	"github.com/fnrun/fnrun/pkg/fn"
)

// Middleware represents an object that can transform input and output values
// when interacting with an Fn.
type Middleware interface {
	Invoke(ctx context.Context, input interface{}, f fn.Fn) (interface{}, error)
}

// Source represents an object that can generate and send inputs to an Fn.
type Source interface {
	Serve(context.Context, fn.Fn) error
}
