// Package timeout provides a middleware that adds a timeout to the context.
//
// The middleware should be configured with a string that can be parsed as a
// time.Duration value. Defaults to 30s.
package timeout

import (
	"context"
	"time"

	"github.com/fnrun/fnrun/pkg/fn"
	"github.com/fnrun/fnrun/pkg/run"
)

type timeoutMiddleware struct {
	duration time.Duration
}

func (t *timeoutMiddleware) ConfigureString(durationStr string) error {
	d, err := time.ParseDuration(durationStr)
	if err != nil {
		return err
	}

	t.duration = d
	return nil
}

func (t *timeoutMiddleware) Invoke(ctx context.Context, input interface{}, f fn.Fn) (interface{}, error) {
	newCtx, cancel := context.WithTimeout(ctx, t.duration)
	defer cancel()

	return f.Invoke(newCtx, input)
}

// New returns a middleware that applies a 30s timeout to the context. The
// duration may be changed via configuration.
func New() run.Middleware {
	return NewWithDuration(30 * time.Second)
}

// NewWithDuration returns a middleware with that applies a timeout specified
// by duration to the context.
func NewWithDuration(duration time.Duration) run.Middleware {
	return &timeoutMiddleware{
		duration: duration,
	}
}
