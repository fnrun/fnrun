package http

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/fnrun/fnrun/pkg/fn"
)

func TestHandler_outputAsBody(t *testing.T) {
	req, err := http.NewRequest("POST", "/", strings.NewReader("some input"))
	if err != nil {
		t.Fatalf("NewRequest returned error: %#v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	rr := httptest.NewRecorder()
	f := fn.NewFnFromInvokeFunc(func(context.Context, interface{}) (interface{}, error) {
		return "output", nil
	})

	config := &httpSourceConfig{
		Addr:              ":8080",
		TreatOutputAsBody: true,
	}

	handler := http.HandlerFunc(makeHandler(ctx, f, config))

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned non-OK status code: %d", rr.Code)
	}

	want := "output"
	got := rr.Body.String()

	if got != want {
		t.Errorf("response output: want %q, got %q", want, got)
	}
}

func TestHandler_outputAsResponse(t *testing.T) {
	req, err := http.NewRequest("POST", "/", strings.NewReader("some input"))
	if err != nil {
		t.Fatalf("NewRequest returned error: %#v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	rr := httptest.NewRecorder()
	f := fn.NewFnFromInvokeFunc(func(context.Context, interface{}) (interface{}, error) {
		return map[string]interface{}{
			"body": `{"a": "b"}`,
			"headers": map[string]string{
				"content-type":    "application/json",
				"x-custom-header": "custom header value",
			},
			"statusCode": http.StatusCreated,
		}, nil
	})

	config := &httpSourceConfig{
		Addr: ":8080",
	}

	handler := http.HandlerFunc(makeHandler(ctx, f, config))

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusCreated {
		t.Errorf("handler returned non-OK status code: %d", rr.Code)
	}

	want := `{"a": "b"}`
	got := rr.Body.String()

	if got != want {
		t.Errorf("response output: want %q, got %q", want, got)
	}

	gotHeaders := make(map[string][]string)
	for k, v := range rr.Result().Header {
		gotHeaders[k] = v
	}
	wantHeaders := map[string][]string{
		"Content-Type":    {"application/json"},
		"X-Custom-Header": {"custom header value"},
	}

	if !reflect.DeepEqual(gotHeaders, wantHeaders) {
		t.Errorf("headers did not match:\nwant: %s\ngot:  %s\n", wantHeaders, gotHeaders)
	}
}

func TestHandler_ignoreOutput(t *testing.T) {
	req, err := http.NewRequest("POST", "/", strings.NewReader("some input"))
	if err != nil {
		t.Fatalf("NewRequest returned error: %#v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	rr := httptest.NewRecorder()
	f := fn.NewFnFromInvokeFunc(func(context.Context, interface{}) (interface{}, error) {
		return map[string]interface{}{
			"body": `{"a": "b"}`,
			"headers": map[string]string{
				"content-type":    "application/json",
				"x-custom-header": "custom header value",
			},
			"statusCode": http.StatusCreated,
		}, nil
	})

	config := &httpSourceConfig{
		Addr:         ":8080",
		IgnoreOutput: true,
	}

	handler := http.HandlerFunc(makeHandler(ctx, f, config))

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusCreated {
		t.Errorf("handler returned non-OK status code: %d", rr.Code)
	}

	want := ""
	got := rr.Body.String()

	if got != want {
		t.Errorf("response output: want %q, got %q", want, got)
	}
}

func TestHandler_defaultHeaders(t *testing.T) {
	req, err := http.NewRequest("POST", "/", strings.NewReader("some input"))
	if err != nil {
		t.Fatalf("NewRequest returned error: %#v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	rr := httptest.NewRecorder()
	f := fn.NewFnFromInvokeFunc(func(context.Context, interface{}) (interface{}, error) {
		return map[string]interface{}{
			"body":       `{"a": "b"}`,
			"statusCode": http.StatusCreated,
		}, nil
	})

	config := &httpSourceConfig{
		Addr: ":8080",
		DefaultHeaders: map[string]string{
			"x-custom-header": "some value",
		},
	}

	handler := http.HandlerFunc(makeHandler(ctx, f, config))

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusCreated {
		t.Errorf("handler returned non-OK status code: %d", rr.Code)
	}

	want := `{"a": "b"}`
	got := rr.Body.String()

	if got != want {
		t.Errorf("response output: want %q, got %q", want, got)
	}

	wantHeaders := map[string][]string{"X-Custom-Header": {"some value"}}
	gotHeaders := make(map[string][]string)
	for k, v := range rr.Result().Header {
		gotHeaders[k] = v
	}

	if !reflect.DeepEqual(gotHeaders, wantHeaders) {
		t.Errorf("headers did not match\nwant: %s\ngot:  %s\n", wantHeaders, gotHeaders)
	}
}

func TestHandler_defaultHeadersWithOutputAsBody(t *testing.T) {
	req, err := http.NewRequest("POST", "/", strings.NewReader("some input"))
	if err != nil {
		t.Fatalf("NewRequest returned error: %#v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	rr := httptest.NewRecorder()
	f := fn.NewFnFromInvokeFunc(func(context.Context, interface{}) (interface{}, error) {
		return "fn output", nil
	})

	config := &httpSourceConfig{
		Addr: ":8080",
		DefaultHeaders: map[string]string{
			"content-type": "application/json",
		},
		TreatOutputAsBody: true,
	}

	handler := http.HandlerFunc(makeHandler(ctx, f, config))

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned non-OK status code: %d", rr.Code)
	}

	want := "fn output"
	got := rr.Body.String()

	if got != want {
		t.Errorf("response output: want %q, got %q", want, got)
	}

	wantHeaders := map[string][]string{"Content-Type": {"application/json"}}
	gotHeaders := make(map[string][]string)
	for k, v := range rr.Result().Header {
		gotHeaders[k] = v
	}

	if !reflect.DeepEqual(gotHeaders, wantHeaders) {
		t.Errorf("headers did not match\nwant: %s\ngot:  %s\n", wantHeaders, gotHeaders)
	}
}

func TestHandler_treatOutputAsBodyWithInteger(t *testing.T) {
	req, err := http.NewRequest("POST", "/", strings.NewReader("some input"))
	if err != nil {
		t.Fatalf("NewRequest returned error: %#v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	rr := httptest.NewRecorder()
	f := fn.NewFnFromInvokeFunc(func(context.Context, interface{}) (interface{}, error) {
		return 1, nil
	})

	config := &httpSourceConfig{
		Addr:              ":8080",
		TreatOutputAsBody: true,
	}

	handler := http.HandlerFunc(makeHandler(ctx, f, config))

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned non-OK status code: %d", rr.Code)
	}

	want := "1"
	got := rr.Body.String()

	if got != want {
		t.Errorf("response output: want %q, got %q", want, got)
	}
}

func TestHandler_error(t *testing.T) {
	req, err := http.NewRequest("POST", "/", strings.NewReader("some input"))
	if err != nil {
		t.Fatalf("NewRequest returned error: %#v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	rr := httptest.NewRecorder()
	f := fn.NewFnFromInvokeFunc(func(context.Context, interface{}) (interface{}, error) {
		return nil, errors.New("some error")
	})

	config := &httpSourceConfig{}

	handler := http.HandlerFunc(makeHandler(ctx, f, config))

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusInternalServerError {
		t.Errorf("return code: want %d, got %d", http.StatusInternalServerError, status)
	}
}

func TestHandler_invalidReturnType(t *testing.T) {
	req, err := http.NewRequest("POST", "/", strings.NewReader("some input"))
	if err != nil {
		t.Fatalf("NewRequest returned error: %#v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	rr := httptest.NewRecorder()
	f := fn.NewFnFromInvokeFunc(func(context.Context, interface{}) (interface{}, error) {
		return 1234, nil
	})

	config := &httpSourceConfig{}

	handler := http.HandlerFunc(makeHandler(ctx, f, config))

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusInternalServerError {
		t.Errorf("return code: want %d, got %d", http.StatusInternalServerError, status)
	}
}
