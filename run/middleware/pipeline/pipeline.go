// Package pipeline provides a middleware that composes several middleware into
// a single pipeline.
//
// The pipeline should be configured with an array of middleware configurations.
// The pipeline will create and configure each middleware before combining them
// into a single middleware.
//
//  If no middleware are defined, the pipeline simply invokes the Fn and returns
// its output.
package pipeline

import (
	"context"
	"errors"
	"fmt"

	"github.com/fnrun/fnrun/fn"
	"github.com/fnrun/fnrun/run"
	"github.com/fnrun/fnrun/run/config"
	"github.com/fnrun/fnrun/run/fn/middleware"
)

var errSingleKey = errors.New("middleware config should be object with single key")

type identityMiddleware struct{}

func (*identityMiddleware) Invoke(ctx context.Context, input interface{}, f fn.Fn) (interface{}, error) {
	return f.Invoke(ctx, input)
}

type composedMiddleware struct {
	a run.Middleware
	b run.Middleware
}

func (wm *composedMiddleware) Invoke(ctx context.Context, input interface{}, f fn.Fn) (interface{}, error) {
	return wm.a.Invoke(ctx, input, middleware.New(wm.b, f))
}

func compose(middlewares ...run.Middleware) run.Middleware {
	var current run.Middleware = &identityMiddleware{}

	for i := len(middlewares) - 1; i >= 0; i-- {
		current = &composedMiddleware{a: middlewares[i], b: current}
	}

	return current
}

type pipelineMiddleware struct {
	middleware run.Middleware
	registry   run.Registry
}

func (p *pipelineMiddleware) Invoke(ctx context.Context, input interface{}, f fn.Fn) (interface{}, error) {
	return p.middleware.Invoke(ctx, input, f)
}

func (p *pipelineMiddleware) ConfigureArray(cfg []interface{}) error {
	middlewares := make([]run.Middleware, 0)

	for _, middlewareConfig := range cfg {
		switch middlewareConfig := middlewareConfig.(type) {
		case string:
			factory, exists := p.registry.FindMiddleware(middlewareConfig)
			if !exists {
				return fmt.Errorf("no middleware registered with key %s", middlewareConfig)
			}
			middleware := factory()
			err := config.Configure(middleware, nil)
			if err != nil {
				return err
			}
			middlewares = append(middlewares, middleware)

		case map[string]interface{}:
			mapConfig := middlewareConfig
			if len(mapConfig) != 1 {
				return errSingleKey
			}
			key := ""
			for k := range mapConfig {
				key = k
			}
			factory, exists := p.registry.FindMiddleware(key)
			if !exists {
				return fmt.Errorf("no middleware registered with key %s", key)
			}
			middleware := factory()
			err := config.Configure(middleware, mapConfig[key])
			if err != nil {
				return err
			}
			middlewares = append(middlewares, middleware)

		default:
			return fmt.Errorf("wrong middleware configuration type: %T, expected string or object", middlewareConfig)
		}
	}

	p.middleware = compose(middlewares...)
	return nil
}

// NewWithRegistry creates a pipeline middleware with registry. The middleware
// has no behavior unless configured.
func NewWithRegistry(registry run.Registry) run.Middleware {
	return &pipelineMiddleware{
		middleware: &identityMiddleware{},
		registry:   registry,
	}
}
