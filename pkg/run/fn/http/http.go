// Package http provides an http fn. The http fn will invoke a remote HTTP
// endpoint and return the response body as output. The fn considers 2xx and 3xx
// HTTP status codes on the response as successfully invocations, and other
// status codes as errors.
package http

import (
	"bytes"
	"context"
	"errors"
	"io/ioutil"
	"net/http"

	"github.com/fnrun/fnrun/pkg/fn"
	"github.com/mitchellh/mapstructure"
)

type httpFn struct {
	config *httpFnConfig
}

type httpFnConfig struct {
	TargetURL   string `mapstructure:"targetURL"`
	ContentType string `mapstructure:"contentType,omitempty"`
}

func (h *httpFn) Invoke(ctx context.Context, input interface{}) (interface{}, error) {
	body, isString := input.(string)
	if !isString {
		return nil, errors.New("expected input to be a string")
	}

	resp, err := http.Post(h.config.TargetURL, h.config.ContentType, bytes.NewBuffer([]byte(body)))

	if err != nil {
		return nil, err
	}

	outputBytes, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		return nil, err
	}

	output := string(outputBytes)

	if resp.StatusCode >= 400 {
		return nil, errors.New(output)
	}
	return output, nil
}

func (*httpFn) RequiresConfig() bool {
	return true
}

func (h *httpFn) ConfigureMap(configMap map[string]interface{}) error {
	return mapstructure.Decode(configMap, h.config)
}

// New returns an http fn with default values. The result of this must be
// configured with a target URL. If a target URL is not configured, calls to
// Invoke will fail.
func New() fn.Fn {
	return &httpFn{
		config: &httpFnConfig{
			ContentType: "application/json",
		},
	}
}
