// Package runner provides a generic runner that instantiates and creates a
// processing pipeline containing a source, a list of middleware, and an fn.
package runner

import (
	"context"
	"errors"

	"github.com/fnrun/fnrun/fn"
	"github.com/fnrun/fnrun/run"
	"github.com/fnrun/fnrun/run/config"
	"github.com/fnrun/fnrun/run/fn/middleware"
	"github.com/mitchellh/mapstructure"
)

// Runner represents a processing pipeline comprising a source, middleware, and
// an fn.
type Runner struct {
	registry run.Registry
	source   run.Source
	fn       fn.Fn
}

// Run executes the processing pipeline.
func (r *Runner) Run(ctx context.Context) error {
	return r.source.Serve(ctx, r.fn)
}

// ConfigureMap configures the runner with source, middleware, and fn values.
func (r *Runner) ConfigureMap(configMap map[string]interface{}) error {
	cfg := struct {
		Source     interface{} `mapstructure:"source"`
		Middleware interface{} `mapstructure:"middleware"`
		Fn         interface{} `mapstructure:"fn"`
	}{}

	mapstructure.Decode(configMap, &cfg)

	if cfg.Source == nil {
		return errors.New("source is a required configuration key")
	}
	if cfg.Fn == nil {
		return errors.New("fn is a required configuration key")
	}

	fnFactory, exists := r.registry.FindFn("fn")
	if !exists {
		return errors.New(`a registered fn not found for key "fn"`)
	}
	f := fnFactory()
	if err := config.Configure(f, cfg.Fn); err != nil {
		return err
	}

	if cfg.Middleware != nil {
		middlewareFactory, exists := r.registry.FindMiddleware("middleware")
		if !exists {
			return errors.New(`a registered middleware not found for key "middleware"`)
		}
		m := middlewareFactory()
		if err := config.Configure(m, cfg.Middleware); err != nil {
			return err
		}
		f = middleware.New(m, f)
	}

	sourceFactory, exists := r.registry.FindSource("source")
	if !exists {
		return errors.New(`a registered source not found for key "source"`)
	}
	source := sourceFactory()
	if err := config.Configure(source, cfg.Source); err != nil {
		return err
	}

	r.fn = f
	r.source = source

	return nil
}

// RequiresConfig always returns true. This method exists to interoperate with
// the config package.
func (r *Runner) RequiresConfig() bool {
	return true
}

// New returns a new instance of a Runner with the specified registry.
func New(registry run.Registry) *Runner {
	return &Runner{registry: registry}
}
