// Package json provides a middleware that performs JSON serialization and
// deserialization.
package json

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/fnrun/fnrun/pkg/fn"
	"github.com/fnrun/fnrun/pkg/run"
	"github.com/mitchellh/mapstructure"
)

type jsonMiddleware struct {
	config *jsonMiddlewareConfig
}

type jsonMiddlewareConfig struct {
	Input  string `mapstructure:"input,omitempty"`
	Output string `mapstructure:"output,omitempty"`
}

func transcode(v interface{}, strategy string) (interface{}, error) {
	newValue := v

	switch strategy {
	case "serialize":
		bytes, err := json.Marshal(v)
		if err != nil {
			return nil, err
		}
		newValue = string(bytes)
		break

	case "deserialize":
		str, ok := v.(string)
		if !ok {
			return nil, fmt.Errorf("expected string value but received one of type %T", v)
		}
		json.Unmarshal([]byte(str), &newValue)
		break
	}

	return newValue, nil
}

func (jm *jsonMiddleware) Invoke(ctx context.Context, input interface{}, f fn.Fn) (interface{}, error) {
	newInput, err := transcode(input, jm.config.Input)

	output, err := f.Invoke(ctx, newInput)
	if err != nil {
		return nil, err
	}

	return transcode(output, jm.config.Output)
}

func (jm *jsonMiddleware) ConfigureMap(configMap map[string]interface{}) error {
	return mapstructure.Decode(configMap, jm.config)
}

// New returns a json middleware that does not manipulate input or output.
func New() run.Middleware {
	return &jsonMiddleware{
		config: &jsonMiddlewareConfig{},
	}
}
