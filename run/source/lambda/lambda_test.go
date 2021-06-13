package lambda_test

import (
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/fnrun/fnrun/fn"
	"github.com/fnrun/fnrun/run/config"
	"github.com/fnrun/fnrun/run/fn/identity"
	"github.com/fnrun/fnrun/run/source/lambda"
)

func TestServe_withDoneContext(t *testing.T) {
	src := lambda.New()
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately before we call serve...

	err := src.Serve(ctx, identity.New())

	got := err
	want := ctx.Err()

	if got != want {
		t.Errorf("unexpected error: want %+v, got %+v", want, got)
	}
}

func TestServe_nextURLReturnsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.WriteHeader(http.StatusInternalServerError)
	}))
	server.Close() // close now so we can trigger the error

	src := lambda.New()
	err := config.Configure(src, map[string]interface{}{
		"runtimeAPI": strings.Replace(server.URL, "http://", "", 1),
	})
	if err != nil {
		t.Fatalf("config.Configure returned error: %+v", err)
	}

	err = src.Serve(context.Background(), identity.New())

	want := "refused"
	got := err.Error()

	if !strings.Contains(got, want) {
		t.Errorf("unexpected error message: want to contain %q, got %q", want, got)
	}
}

func TestServe_createsCorrectInput(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.Header().Add("Lambda-Runtime-Aws-Request-Id", "1")
		rw.Header().Add("Lambda-Runtime-Deadline-Ms", "500")
		rw.Header().Add("Lambda-Runtime-Invoked-Function-Arn", "some arn")
		rw.Header().Add("Lambda-Runtime-Trace-Id", "trace-id")
		rw.Header().Add("Content-Type", "application/json")
		rw.WriteHeader(http.StatusOK)
		rw.Write([]byte(`{"key": "value"}`))
	}))
	defer server.Close()

	src := lambda.New()
	err := config.Configure(src, map[string]interface{}{
		"runtimeAPI": strings.Replace(server.URL, "http://", "", 1),
	})
	if err != nil {
		t.Fatalf("config.Configure returned error: %+v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	inputCh := make(chan interface{}, 1)

	f := fn.NewFnFromInvokeFunc(func(ctx context.Context, input interface{}) (interface{}, error) {
		inputCh <- input
		return input, nil
	})

	go func() {
		src.Serve(ctx, f)
	}()

	select {
	case <-ctx.Done():
		t.Fatal("function not called within the timeout period")
	case got := <-inputCh:
		want := map[string]interface{}{
			"LambdaRuntimeAwsRequestId":       "1",
			"LambdaRuntimeDeadlineMs":         int64(500),
			"LambdaRuntimeInvokedFunctionArn": "some arn",
			"LambdaRuntimeTraceId":            "trace-id",
			"event":                           map[string]interface{}{"key": "value"},
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("unexpected input:\nwant %#v\ngot  %#v", want, got)
		}
	}
}

func TestServe_createsCorrectInput_noEventDeserialization(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.Header().Add("Lambda-Runtime-Aws-Request-Id", "1")
		rw.Header().Add("Lambda-Runtime-Deadline-Ms", "500")
		rw.Header().Add("Lambda-Runtime-Invoked-Function-Arn", "some arn")
		rw.Header().Add("Lambda-Runtime-Trace-Id", "trace-id")
		rw.Header().Add("Content-Type", "application/json")
		rw.WriteHeader(http.StatusOK)
		rw.Write([]byte(`{"key": "value"}`))
	}))
	defer server.Close()

	src := lambda.New()
	err := config.Configure(src, map[string]interface{}{
		"jsonDeserializeEvent": false,
		"runtimeAPI":           strings.Replace(server.URL, "http://", "", 1),
	})
	if err != nil {
		t.Fatalf("config.Configure returned error: %+v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	inputCh := make(chan interface{}, 1)

	f := fn.NewFnFromInvokeFunc(func(ctx context.Context, input interface{}) (interface{}, error) {
		inputCh <- input
		return input, nil
	})

	go func() {
		src.Serve(ctx, f)
	}()

	select {
	case <-ctx.Done():
		t.Fatal("function not called within the timeout period")
	case got := <-inputCh:
		want := map[string]interface{}{
			"LambdaRuntimeAwsRequestId":       "1",
			"LambdaRuntimeDeadlineMs":         int64(500),
			"LambdaRuntimeInvokedFunctionArn": "some arn",
			"LambdaRuntimeTraceId":            "trace-id",
			"event":                           `{"key": "value"}`,
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("unexpected input:\nwant %#v\ngot  %#v", want, got)
		}
	}
}

func TestServe_postsSuccess(t *testing.T) {
	successCh := make(chan interface{}, 1)

	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		t.Log(req.URL.String())

		switch req.URL.String() {
		case "/2018-06-01/runtime/invocation/next":
			rw.Header().Add("Lambda-Runtime-Aws-Request-Id", "request-id")
			rw.Header().Add("Lambda-Runtime-Deadline-Ms", "500")
			rw.Header().Add("Lambda-Runtime-Invoked-Function-Arn", "some arn")
			rw.Header().Add("Lambda-Runtime-Trace-Id", "trace-id")
			rw.Header().Add("Content-Type", "application/json")
			rw.WriteHeader(http.StatusOK)
			rw.Write([]byte(`{"key": "value"}`))

		case "/2018-06-01/runtime/invocation/request-id/response":
			body, err := ioutil.ReadAll(req.Body)
			if err != nil {
				t.Fatalf("error reading body in success response: %+v", err)
			}
			defer req.Body.Close()
			successCh <- string(body)
			rw.WriteHeader(http.StatusOK)

		default:
			t.Fatalf("unexpected URI: %q", req.URL.String())
		}
	}))
	defer server.Close()

	src := lambda.New()
	err := config.Configure(src, map[string]interface{}{
		"jsonDeserializeEvent": false,
		"runtimeAPI":           strings.Replace(server.URL, "http://", "", 1),
	})
	if err != nil {
		t.Fatalf("config.Configure returned error: %+v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	f := fn.NewFnFromInvokeFunc(func(ctx context.Context, input interface{}) (interface{}, error) {
		return "some output", nil
	})

	go func() {
		src.Serve(ctx, f)
	}()

	select {
	case <-ctx.Done():
		t.Fatal("function not called within the timeout period")
	case got := <-successCh:
		want := "some output"
		if !reflect.DeepEqual(got, want) {
			t.Errorf("unexpected input:\nwant %#v\ngot  %#v", want, got)
		}
	}
}

func TestServe_postsError(t *testing.T) {
	errorCh := make(chan interface{}, 1)

	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		t.Log(req.URL.String())

		switch req.URL.String() {
		case "/2018-06-01/runtime/invocation/next":
			rw.Header().Add("Lambda-Runtime-Aws-Request-Id", "request-id")
			rw.Header().Add("Lambda-Runtime-Deadline-Ms", "500")
			rw.Header().Add("Lambda-Runtime-Invoked-Function-Arn", "some arn")
			rw.Header().Add("Lambda-Runtime-Trace-Id", "trace-id")
			rw.Header().Add("Content-Type", "application/json")
			rw.WriteHeader(http.StatusOK)
			rw.Write([]byte(`{"key": "value"}`))

		case "/2018-06-01/runtime/invocation/request-id/error":
			body, err := ioutil.ReadAll(req.Body)
			if err != nil {
				t.Fatalf("error reading body in success response: %+v", err)
			}
			defer req.Body.Close()

			var errorMessage map[string]interface{}
			err = json.Unmarshal(body, &errorMessage)
			if err != nil {
				t.Fatalf("error unmarshaling error message: %+v", err)
			}
			errorCh <- errorMessage
			rw.WriteHeader(http.StatusOK)

		default:
			t.Fatalf("unexpected URI: %q", req.URL.String())
		}
	}))
	defer server.Close()

	src := lambda.New()
	err := config.Configure(src, map[string]interface{}{
		"jsonDeserializeEvent": false,
		"runtimeAPI":           strings.Replace(server.URL, "http://", "", 1),
	})
	if err != nil {
		t.Fatalf("config.Configure returned error: %+v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	f := fn.NewFnFromInvokeFunc(func(ctx context.Context, input interface{}) (interface{}, error) {
		return nil, errors.New("some error")
	})

	go func() {
		src.Serve(ctx, f)
	}()

	select {
	case <-ctx.Done():
		t.Fatal("function not called within the timeout period")
	case got := <-errorCh:
		want := map[string]interface{}{
			"errorMessage": "some error",
			"errorType":    "FunctionExecutionError",
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("unexpected input:\nwant %#v\ngot  %#v", want, got)
		}
	}
}

func TestServe(t *testing.T) {
	// make a server that will requests with incrementing ids
	// it will send success and error posts to a receive channel
}
