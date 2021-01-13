package fn

import "context"

// Fn represents a function that encapsulates some application functionality.
//
// Fn implementations may contain the functionality directly or facilitate some
// process by which the functionality is invoked.
type Fn interface {
	Invoke(context.Context, interface{}) (interface{}, error)
}

type InvokeFunc func(context.Context, interface{}) (interface{}, error)

type invokeFuncFn struct {
	f InvokeFunc
}

func (i *invokeFuncFn) Invoke(ctx context.Context, input interface{}) (interface{}, error) {
	return i.f(ctx, input)
}

func NewFnFromInvokeFunc(i InvokeFunc) Fn {
	return &invokeFuncFn{f: i}
}
