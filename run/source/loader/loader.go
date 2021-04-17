package loader

import (
	"context"
	"fmt"

	"github.com/fnrun/fnrun/fn"
	"github.com/fnrun/fnrun/run"
	"github.com/fnrun/fnrun/run/config"
)

type wrappedSource struct {
	source   run.Source
	registry run.Registry
}

func (w *wrappedSource) configure(sourceKey string, sourceConfig interface{}) error {
	sourceFactory, exists := w.registry.FindSource(sourceKey)
	if !exists {
		return fmt.Errorf("a registered source not found for key %q", sourceKey)
	}

	source := sourceFactory()
	if err := config.Configure(source, sourceConfig); err != nil {
		return err
	}

	w.source = source
	return nil
}

func (w *wrappedSource) ConfigureString(sourceKey string) error {
	return w.configure(sourceKey, nil)
}

func (w *wrappedSource) ConfigureMap(configMap map[string]interface{}) error {
	sourceKey, sourceConfig, err := config.GetSinglePair(configMap)
	if err != nil {
		return err
	}

	return w.configure(sourceKey, sourceConfig)
}

func (w *wrappedSource) RequiresConfig() bool {
	return true
}

func (w *wrappedSource) Serve(ctx context.Context, f fn.Fn) error {
	return w.source.Serve(ctx, f)
}

// New returns a source that can instantiate another source based on
// its configuration data and information held in the registry.
func New(registry run.Registry) run.Source {
	return &wrappedSource{registry: registry}
}
