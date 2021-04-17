// Package ratelimiter provides a middleware that restricts the frequency of
// calls to a specified rate with a maximum burst.
//
// The default value is a ratelimiter that creates one token per second with a
// burst of one.
package ratelimiter

import (
	"context"
	"sync"
	"time"

	"github.com/fnrun/fnrun/fn"
	"github.com/fnrun/fnrun/run"
	"github.com/mitchellh/mapstructure"
	"golang.org/x/time/rate"
)

type rlmiddleware struct {
	Burst int           `mapstructure:",omitempty"`
	Every time.Duration `mapstructure:",omitempty"`

	limiter *rate.Limiter
	once    sync.Once
}

func (rl *rlmiddleware) ConfigureMap(configMap map[string]interface{}) error {
	decoderConfig := &mapstructure.DecoderConfig{
		Metadata: nil,
		Result:   rl,
		DecodeHook: mapstructure.ComposeDecodeHookFunc(
			mapstructure.StringToTimeDurationHookFunc(),
		),
	}
	decoder, _ := mapstructure.NewDecoder(decoderConfig)

	return decoder.Decode(configMap)
}

func (rl *rlmiddleware) Invoke(ctx context.Context, input interface{}, f fn.Fn) (interface{}, error) {
	rl.once.Do(func() {
		rl.limiter = rate.NewLimiter(rate.Every(rl.Every), rl.Burst)
	})

	if err := rl.limiter.Wait(ctx); err != nil {
		return nil, err
	}

	return f.Invoke(ctx, input)
}

// New returns a ratelimiter middleware configured to allow one token to be
// generated every second with a burst of one.
func New() run.Middleware {
	return &rlmiddleware{
		Burst: 1,
		Every: 1 * time.Second,
	}
}
