// Package debug provides a middleware that prints inputs, outputs, and errors
// to stdout.
//
// The middleware may be configured with a boolean value that describes whether
// it should print, which defaults to true.
package debug

import (
	"context"
	"log"

	"github.com/fnrun/fnrun/fn"
	"github.com/fnrun/fnrun/run"
)

type debugMiddleware struct {
	PrintEnabled bool
}

func (d *debugMiddleware) ConfigureBool(printEnabled bool) error {
	d.PrintEnabled = printEnabled
	return nil
}

func (d *debugMiddleware) Invoke(ctx context.Context, input interface{}, f fn.Fn) (interface{}, error) {
	if d.PrintEnabled {
		log.Printf("debugMiddleware: Handling input %#v", input)
	}

	output, err := f.Invoke(ctx, input)

	if d.PrintEnabled {
		log.Printf("debugMiddleware: Received output %#v", output)
		if err != nil {
			log.Printf("debugMiddleware: Received error %q", err)
		}
	}

	return output, err
}

// New returns a debug middleware with printing enabled. The value may be
// configured with a bool to indicate whether printing should be enabled
// explicitly.
func New() run.Middleware {
	return NewWithPrintEnabled(true)
}

// NewWithPrintEnabled returns a debug middleware configured with printEnabled.
func NewWithPrintEnabled(printEnabled bool) run.Middleware {
	return &debugMiddleware{
		PrintEnabled: printEnabled,
	}
}
