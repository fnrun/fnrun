package lambda

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"

	"github.com/fnrun/fnrun/pkg/fn"
	"github.com/fnrun/fnrun/pkg/run"
	"github.com/mitchellh/mapstructure"
)

type lambdaSource struct {
	config *lambdaSourceConfig
}

type lambdaSourceConfig struct {
	JSONDeserializeEvent bool `mapstructure:"jsonDeserializeEvent,omitempty"`
}

func (l *lambdaSource) Serve(ctx context.Context, f fn.Fn) error {
	runtimeAPI := os.Getenv("AWS_LAMBDA_RUNTIME_API")
	baseURL := fmt.Sprintf("http://%s/2018-06-01/runtime/invocation", runtimeAPI)
	nextURL := fmt.Sprintf("%s/next", baseURL)

	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		resp, err := http.Get(nextURL)
		if err != nil {
			return err
		}

		invocationID := resp.Header.Get("Lambda-Runtime-Aws-Request-Id")

		input, err := createInput(resp, l.config)
		if err != nil {
			return err
		}

		output, err := f.Invoke(ctx, input)
		if err != nil {
			if err := postError(baseURL, invocationID, err); err != nil {
				return err
			}
			continue
		}

		responseURL := fmt.Sprintf("%s/%s/response", baseURL, invocationID)

		var responseData []byte

		switch output := output.(type) {
		case string:
			responseData = []byte(output)
		case map[string]interface{}:
			responseData, err = json.Marshal(output)
			if err != nil {
				return err
			}
		default:
			return errors.New("expected output to be either a string or map[string]interface{}")
		}

		http.Post(responseURL, "application/json", bytes.NewBuffer(responseData))
	}
}

func postError(baseURL string, invocationID string, errToSend error) error {
	errorURL := fmt.Sprintf("%s/%s/error", baseURL, invocationID)
	errorData, err := json.Marshal(struct {
		ErrorMessage string `json:"errorMessage"`
		ErrorType    string `json:"errorType"`
	}{
		ErrorMessage: errToSend.Error(),
		ErrorType:    "FunctionExecutionError",
	})
	if err != nil {
		return err
	}

	_, err = http.Post(errorURL, "application/json", bytes.NewBuffer(errorData))
	return err
}

func createInput(resp *http.Response, config *lambdaSourceConfig) (map[string]interface{}, error) {
	input := make(map[string]interface{})
	eventBytes, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		return nil, err
	}

	if requestID := resp.Header.Get("Lambda-Runtime-Aws-Request-Id"); requestID != "" {
		input["LambdaRuntimeAwsRequestId"] = requestID
	}
	if deadlineMs := resp.Header.Get("Lambda-Runtime-Deadline-Ms"); deadlineMs != "" {
		ms, err := strconv.ParseInt(deadlineMs, 10, 64)
		if nil == err {
			input["LambdaRuntimeDeadlineMs"] = ms
		}
	}
	if arn := resp.Header.Get("Lambda-Runtime-Invoked-Function-Arn"); arn != "" {
		input["LambdaRuntimeInvokedFunctionArn"] = arn
	}
	if traceID := resp.Header.Get("Lambda-Runtime-Trace-Id"); traceID != "" {
		input["LambdaRuntimeTraceId"] = traceID
	}

	if config.JSONDeserializeEvent {
		body := make(map[string]interface{})
		if err := json.Unmarshal(eventBytes, &body); err != nil {
			return nil, err
		}
		input["event"] = body
	} else {
		input["event"] = string(eventBytes)
	}

	return input, nil
}

func (l *lambdaSource) ConfigureMap(configMap map[string]interface{}) error {
	return mapstructure.Decode(configMap, l.config)
}

// New returns a source that serves requests from AWS Lambda.
func New() run.Source {
	return &lambdaSource{
		config: &lambdaSourceConfig{
			JSONDeserializeEvent: true,
		},
	}
}
