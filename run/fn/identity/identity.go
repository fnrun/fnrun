// Package identity provides a function that returns its input.
package identity

import (
	"context"

	"github.com/fnrun/fnrun/fn"
)

type identityFn struct{}

func (*identityFn) Invoke(ctx context.Context, input interface{}) (interface{}, error) {
	return input, nil
}

// New returns a function that returns its input.
//
// The identity function is useful for testing or when all processing of an
// input is handled by middleware.
func New() fn.Fn {
	return &identityFn{}
}
