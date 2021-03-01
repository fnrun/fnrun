// Package fn provides the interfaces and functions necessary to create
// functions for fnrun. This is the only package that is necessary for
// developers to use if they are using fnrun as a library to build functions.
package fn

import "context"

// Fn represents a function that encapsulates some application functionality.
//
// Fn implementations may contain the functionality directly or facilitate some
// process by which the functionality is invoked.
type Fn interface {
	Invoke(context.Context, interface{}) (interface{}, error)
}

// InvokeFunc is an adapter to allow the use of ordinary functions as the basis
// for an Fn.
type InvokeFunc func(context.Context, interface{}) (interface{}, error)

type invokeFuncFn struct {
	f InvokeFunc
}

func (i *invokeFuncFn) Invoke(ctx context.Context, input interface{}) (interface{}, error) {
	return i.f(ctx, input)
}

// NewFnFromInvokeFunc wraps an InvokeFunc in an Fn and returns the result.
func NewFnFromInvokeFunc(i InvokeFunc) Fn {
	return &invokeFuncFn{f: i}
}
