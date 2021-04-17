package run

import (
	"context"

	"github.com/fnrun/fnrun/fn"
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

// Registry represents an object that sources and finds factories for sources,
// middlewares, and functions under specified keys.
type Registry interface {
	RegisterSource(key string, f func() Source)
	RegisterMiddleware(key string, f func() Middleware)
	RegisterFn(key string, f func() fn.Fn)

	RegisterSourceWithRegistry(key string, f func(Registry) Source)
	RegisterMiddlewareWithRegistry(key string, f func(Registry) Middleware)
	RegisterFnWithRegistry(key string, f func(Registry) fn.Fn)

	FindSource(key string) (func() Source, bool)
	FindMiddleware(key string) (func() Middleware, bool)
	FindFn(key string) (func() fn.Fn, bool)
}

type registry struct {
	source     map[string]func() Source
	middleware map[string]func() Middleware
	fn         map[string]func() fn.Fn
}

func (r *registry) RegisterSource(key string, f func() Source) {
	r.source[key] = f
}
func (r *registry) RegisterMiddleware(key string, f func() Middleware) {
	r.middleware[key] = f
}
func (r *registry) RegisterFn(key string, f func() fn.Fn) {
	r.fn[key] = f
}

func (r *registry) RegisterSourceWithRegistry(key string, f func(Registry) Source) {
	r.RegisterSource(key, func() Source { return f(r) })
}

func (r *registry) RegisterMiddlewareWithRegistry(key string, f func(Registry) Middleware) {
	r.RegisterMiddleware(key, func() Middleware { return f(r) })
}

func (r *registry) RegisterFnWithRegistry(key string, f func(Registry) fn.Fn) {
	r.RegisterFn(key, func() fn.Fn { return f(r) })
}

func (r *registry) FindSource(key string) (func() Source, bool) {
	f, exists := r.source[key]
	return f, exists
}

func (r *registry) FindMiddleware(key string) (func() Middleware, bool) {
	f, exists := r.middleware[key]
	return f, exists
}

func (r *registry) FindFn(key string) (func() fn.Fn, bool) {
	f, exists := r.fn[key]
	return f, exists
}

// NewRegistry creates a new, empty Registry.
func NewRegistry() Registry {
	return &registry{
		source:     make(map[string]func() Source),
		middleware: make(map[string]func() Middleware),
		fn:         make(map[string]func() fn.Fn),
	}
}
