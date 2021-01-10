// Package pool provides an Fn that contain a pool of Fns. The pool will ensure
// that each Fn in the pool will process only one input at a time. In the case
// that Fns are long-running, the pool will wait for an Fn to become available
// for up to a maximum wait duration which defaults to 500ms. After the wait
// duration has expired, the pool will return an ErrAvailabilityTimeout instead.
//
// Configuration options for this pool are required and should be a map. There
// are three configuration options available:
//
// - concurrency: an int that describes how many Fns will exist in the pool
// - maxWaitDuration: a string that can be parsed into a time.Duration
// - template: a string or map configuration for an Fn
package pool

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/fnrun/fnrun/pkg/fn"
	"github.com/fnrun/fnrun/pkg/run"
	"github.com/fnrun/fnrun/pkg/run/config"
	"github.com/mitchellh/mapstructure"
)

// ErrAvailabilityTimeout is an error that occurs when an Fn was not fetched
// from the pool before a timeout occurred.
var ErrAvailabilityTimeout = errors.New("could not get access to Fn before timeout")

type poolFn struct {
	maxWaitDuration time.Duration
	registry        run.Registry
	fnChan          chan fn.Fn
}

func (p *poolFn) Invoke(ctx context.Context, input interface{}) (interface{}, error) {
	select {
	case f := <-p.fnChan:
		output, err := f.Invoke(ctx, input)
		p.fnChan <- f
		if err != nil {
			return nil, err
		}
		return output, nil
	case <-time.After(p.maxWaitDuration):
		return nil, ErrAvailabilityTimeout
	}
}

func (*poolFn) RequiresConfig() bool {
	return true
}

func createFnFromString(r run.Registry, fnName string) (fn.Fn, error) {
	factory, hasFn := r.FindFn(fnName)
	if !hasFn {
		return nil, fmt.Errorf("fn %s was not registered with r", fnName)
	}
	f := factory()
	return f, config.Configure(f, nil)
}

func createFnFromTemplate(r run.Registry, config interface{}) (fn.Fn, error) {
	switch config.(type) {
	case string:
		return createFnFromString(r, config.(string))
	default:
		return nil, errors.New("unsupported config type")
	}
}

func (p *poolFn) ConfigureMap(configMap map[string]interface{}) error {
	cfg := struct {
		MaxWaitDuration string      `mapstructure:"maxWaitDuration"`
		Concurrency     int         `mapstructure:"concurrency"`
		Template        interface{} `mapstructure:"template"`
	}{}

	err := mapstructure.Decode(configMap, &cfg)
	if err != nil {
		return err
	}

	if cfg.MaxWaitDuration != "" {
		d, err := time.ParseDuration(cfg.MaxWaitDuration)
		if err != nil {
			return err
		}

		p.maxWaitDuration = d
	}

	concurrency := cfg.Concurrency
	if concurrency == 0 {
		concurrency = 8
	}

	p.fnChan = make(chan fn.Fn, concurrency)

	for i := 0; i < concurrency; i++ {
		f, err := createFnFromTemplate(p.registry, cfg.Template)
		if err != nil {
			return err
		}
		p.fnChan <- f
	}

	return nil
}

// New creates a new Fn pool that must be configured before use.
func New(registry run.Registry) fn.Fn {
	return &poolFn{
		registry:        registry,
		maxWaitDuration: 500 * time.Millisecond,
	}
}
