// Package loader provides an Fn implementation that can be configured with a
// function configuration and registry.
package loader

import (
	"context"
	"fmt"

	"github.com/fnrun/fnrun/pkg/fn"
	"github.com/fnrun/fnrun/pkg/run"
	"github.com/fnrun/fnrun/pkg/run/config"
)

type wrappedFn struct {
	fn       fn.Fn
	registry run.Registry
}

func (w *wrappedFn) ConfigureString(fnKey string) error {
	fnFactory, exists := w.registry.FindFn(fnKey)
	if !exists {
		return fmt.Errorf("a registered function not found for key %q", fnKey)
	}
	fn := fnFactory()
	err := config.Configure(fn, nil)
	if err != nil {
		return err
	}

	w.fn = fn
	return err
}

func (w *wrappedFn) ConfigureMap(configMap map[string]interface{}) error {
	fnKey, fnConfig, err := config.GetSinglePair(configMap)
	if err != nil {
		return err
	}

	fnFactory, exists := w.registry.FindFn(fnKey)
	if !exists {
		return fmt.Errorf("a registered function not found for key %q", fnKey)
	}

	fn := fnFactory()
	if err := config.Configure(fn, fnConfig); err != nil {
		return err
	}

	w.fn = fn
	return nil
}

func (w *wrappedFn) RequiresConfig() bool {
	return true
}

func (wf *wrappedFn) Invoke(ctx context.Context, input interface{}) (interface{}, error) {
	return wf.fn.Invoke(ctx, input)
}

// New creates a configurable Fn that can be configured with a string or map
// configuration.
func New(registry run.Registry) fn.Fn {
	return &wrappedFn{registry: registry}
}
