// Package circuitbreaker provides a circuit breaking middleware based on
// sony/gobreaker.
package circuitbreaker

import (
	"context"
	"time"

	"github.com/fnrun/fnrun/pkg/fn"
	"github.com/fnrun/fnrun/pkg/run"
	"github.com/mitchellh/mapstructure"
	"github.com/sony/gobreaker"
)

type circuitBreakerMiddleware struct {
	cb *gobreaker.CircuitBreaker
}

type breakerConfig struct {
	Name        string        `mapstructure:"name,omitempty"`
	MaxRequests uint32        `mapstructure:"maxRequests,omitempty"`
	Interval    time.Duration `mapstructure:"interval,omitempty"`
	Timeout     time.Duration `mapstructure:"timeout,omitempty"`
}

func (c *breakerConfig) toSettings() gobreaker.Settings {
	return gobreaker.Settings{
		Name:        c.Name,
		MaxRequests: c.MaxRequests,
		Interval:    c.Interval,
		Timeout:     c.Timeout,
	}
}

func (cbm *circuitBreakerMiddleware) Invoke(ctx context.Context, input interface{}, f fn.Fn) (interface{}, error) {
	return cbm.cb.Execute(func() (interface{}, error) {
		return f.Invoke(ctx, input)
	})
}

func (cbm *circuitBreakerMiddleware) ConfigureMap(configMap map[string]interface{}) error {
	cfg := &breakerConfig{}
	mapstructure.Decode(configMap, cfg)

	cbm.cb = gobreaker.NewCircuitBreaker(cfg.toSettings())

	return nil
}

func New() run.Middleware {
	return &circuitBreakerMiddleware{}
}
