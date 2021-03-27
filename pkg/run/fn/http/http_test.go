package http_test

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fnrun/fnrun/pkg/run/config"
	httpfn "github.com/fnrun/fnrun/pkg/run/fn/http"
)

func TestConfigure_invalidValue(t *testing.T) {
	f := httpfn.New()
	err := config.Configure(f, nil)

	if err == nil {
		t.Fatal("expected config.Configure to return error but it did not")
	}
}

func TestInvoke_invalidInput(t *testing.T) {
	f := httpfn.New()
	output, err := f.Invoke(context.Background(), map[string]interface{}{})
	if err == nil {
		t.Fatalf("expected Invoke to return an error but it did not")
	}

	if output != nil {
		t.Errorf("expected output to be nil but was %#v", output)
	}

	got := err.Error()
	want := "expected input to be a string"

	if got != want {
		t.Errorf("unexpected error message: want %q, got %q", want, got)
	}
}

func TestInvoke_clientErrorReturnsError(t *testing.T) {
	f := httpfn.New()
	output, err := f.Invoke(context.Background(), "some input")
	if err == nil {
		t.Fatal("expected Invoke to return an error but it did not")
	}

	if output != nil {
		t.Errorf("expected output to be nil but it was %#v", output)
	}

	got := err.Error()
	want := `Post "": unsupported protocol scheme ""`

	if got != want {
		t.Errorf("unexpected error message: want %q, got %q", want, got)
	}
}

func equals(t *testing.T, got, want interface{}) {
	t.Helper()

	if got != want {
		t.Errorf("want %#v, got %#v", want, got)
	}
}

func TestInvoke_clientReceivesErrorHTTPResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		equals(t, req.URL.String(), "/some/path")

		body, err := ioutil.ReadAll(req.Body)
		if err != nil {
			t.Fatal(err)
		}
		equals(t, string(body), "some input")

		rw.WriteHeader(400)
		rw.Write([]byte("some response"))
	}))
	defer server.Close()

	f := httpfn.New()
	err := config.Configure(f, map[string]interface{}{
		"targetURL": fmt.Sprintf("%s/some/path", server.URL),
	})
	if err != nil {
		t.Fatalf("config.Configure returned error: %+v", err)
	}

	output, err := f.Invoke(context.Background(), "some input")
	if err == nil {
		t.Fatal("expected Invoke to return an error but it did not")
	}

	if output != nil {
		t.Errorf("expected output to be nil but it was %#v", output)
	}

	equals(t, err.Error(), "some response")
}

func TestInvoke_clientReceivesOKResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		equals(t, req.URL.String(), "/some/path")

		body, err := ioutil.ReadAll(req.Body)
		if err != nil {
			t.Fatal(err)
		}
		equals(t, string(body), "some input")

		rw.WriteHeader(200)
		rw.Write([]byte("some response"))
	}))
	defer server.Close()

	f := httpfn.New()
	err := config.Configure(f, map[string]interface{}{
		"targetURL": fmt.Sprintf("%s/some/path", server.URL),
	})
	if err != nil {
		t.Fatalf("config.Configure returned error: %+v", err)
	}

	output, err := f.Invoke(context.Background(), "some input")
	if err != nil {
		t.Errorf("Invoke returned error: %+v", err)
	}

	equals(t, output, "some response")
}
