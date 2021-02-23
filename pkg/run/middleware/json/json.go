// Package json provides a middleware that performs JSON serialization and
// deserialization.
package json

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/fnrun/fnrun/pkg/fn"
	"github.com/fnrun/fnrun/pkg/run"
)

type jsonMiddlewareConfig struct {
	serializeInput    bool
	deserializeOutput bool
}

type jsonMiddleware struct {
	config *jsonMiddlewareConfig
}

func (j *jsonMiddleware) Invoke(ctx context.Context, input interface{}, f fn.Fn) (interface{}, error) {
	newInput := input

	if j.config.serializeInput {
		jsonBytes, err := json.Marshal(input)
		if err != nil {
			return nil, err
		}
		newInput = string(jsonBytes)
	}

	output, err := f.Invoke(ctx, newInput)
	if err != nil {
		return output, err
	}

	if j.config.deserializeOutput {
		var newOutput interface{}
		err := json.Unmarshal([]byte(fmt.Sprint(output)), &newOutput)
		if err != nil {
			return output, err
		}
		output = newOutput
	}

	return output, err
}

func (j *jsonMiddleware) ConfigureString(config string) error {
	switch config {
	case "both":
		j.config.serializeInput = true
		j.config.deserializeOutput = true
		break

	case "input":
		j.config.serializeInput = true
		j.config.deserializeOutput = false
		break

	case "output":
		j.config.serializeInput = false
		j.config.deserializeOutput = true
		break

	default:
		return fmt.Errorf(`unsupported config value: %q; expected "input", "output", or "both"`, config)
	}

	return nil
}

// New returns a middleware that can serialize its input to JSON, deserialize
// its output, or both.
func New() run.Middleware {
	return &jsonMiddleware{
		config: &jsonMiddlewareConfig{
			serializeInput:    true,
			deserializeOutput: true,
		},
	}
}
