// Package jq provides a middleware that provides a jq interface to the input
// and output from a function.
package jq

import (
	"context"

	"github.com/fnrun/fnrun/pkg/fn"
	"github.com/fnrun/fnrun/pkg/run"
	"github.com/itchyny/gojq"
	"github.com/mitchellh/mapstructure"
)

type jqMiddleware struct {
	input  *gojq.Code
	output *gojq.Code
}

type jqMiddlewareConfig struct {
	Input  string `mapstructure:"input"`
	Output string `mapstructure:"output"`
}

func compile(pattern string) (*gojq.Code, error) {
	if pattern == "" {
		return nil, nil
	}

	query, err := gojq.Parse(pattern)
	if err != nil {
		return nil, err
	}

	return gojq.Compile(query)
}

func apply(ctx context.Context, code *gojq.Code, input interface{}) (interface{}, error) {
	if code == nil {
		return input, nil
	}

	var processed []interface{}
	iter := code.RunWithContext(ctx, input)
	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		if err, ok := v.(error); ok {
			return nil, err
		}
		processed = append(processed, v)
	}

	if len(processed) == 1 {
		return processed[0], nil
	}

	return processed, nil
}

func (*jqMiddleware) RequiresConfig() bool {
	return true
}

func (j *jqMiddleware) ConfigureMap(configMap map[string]interface{}) error {
	var config jqMiddlewareConfig
	if err := mapstructure.Decode(configMap, &config); err != nil {
		return err
	}

	inputCode, err := compile(config.Input)
	if err != nil {
		return err
	}
	j.input = inputCode

	outputCode, err := compile(config.Output)
	if err != nil {
		return err
	}
	j.output = outputCode

	return nil
}

func (j *jqMiddleware) Invoke(ctx context.Context, input interface{}, f fn.Fn) (interface{}, error) {
	newInput := input

	newInput, err := apply(ctx, j.input, input)
	if err != nil {
		return nil, err
	}

	output, err := f.Invoke(ctx, newInput)
	if err != nil {
		return output, err
	}

	newOutput, err := apply(ctx, j.output, output)

	return newOutput, nil
}

// New returns a middleware that applies jq patterns to inputs and/or outputs.
func New() run.Middleware {
	return &jqMiddleware{}
}
