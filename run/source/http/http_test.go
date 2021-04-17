package http

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"
	"time"

	"github.com/fnrun/fnrun/fn"
	"github.com/fnrun/fnrun/run/config"
	"github.com/fnrun/fnrun/run/fn/identity"
)

func TestConfigureMap_invalidInput(t *testing.T) {
	src := New()
	err := config.Configure(src, map[string]interface{}{
		"ignoreOutput": 3,
	})

	if err == nil {
		t.Fatal("expected config.Configure to return an error but it did not")
	}
}

func TestServe(t *testing.T) {
	src := New().(*httpSource)
	err := config.Configure(src, map[string]interface{}{
		"address":           "127.0.0.1:0",
		"treatOutputAsBody": true,
		"outputHeaders": map[string]string{
			"Content-Type": "application/json",
		},
	})
	url := fmt.Sprintf("http://%s/", src.Listener.Addr().String())

	if err != nil {
		t.Fatalf("config.Configure returned an error: %+v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		src.Serve(ctx, fn.NewFnFromInvokeFunc(func(ctx context.Context, input interface{}) (interface{}, error) {
			return input.(map[string]interface{})["body"], nil
		}))
	}()

	body := "some value"
	resp, err := http.Post(url, "application/json", bytes.NewBuffer([]byte(body)))
	if err != nil {
		t.Fatalf("error posting: %+v", err)
	}

	respBody, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		t.Fatalf("ioutil.ReadAll returned error: %+v", err)
	}

	want := "some value"
	got := string(respBody)

	if got != want {
		t.Errorf("unexpected result: want %q, got %q", want, got)
	}

	wantContentType := "application/json"
	gotContentType := resp.Header.Get("Content-Type")
	if gotContentType != wantContentType {
		t.Errorf("incorrect content-type: want %q, got %q", wantContentType, gotContentType)
	}
}

func TestServe_ignoreOutput(t *testing.T) {
	src := New().(*httpSource)
	err := config.Configure(src, map[string]interface{}{
		"address":           "127.0.0.1:0",
		"ignoreOutput":      true,
		"treatOutputAsBody": true,
	})
	url := fmt.Sprintf("http://%s/", src.Listener.Addr().String())

	if err != nil {
		t.Fatalf("config.Configure returned an error: %+v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		src.Serve(ctx, fn.NewFnFromInvokeFunc(func(ctx context.Context, input interface{}) (interface{}, error) {
			return input.(map[string]interface{})["body"], nil
		}))
	}()

	body := "some value"
	resp, err := http.Post(url, "application/json", bytes.NewBuffer([]byte(body)))
	if err != nil {
		t.Fatalf("error posting: %+v", err)
	}

	respBody, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		t.Fatalf("ioutil.ReadAll returned error: %+v", err)
	}

	want := ""
	got := string(respBody)

	if got != want {
		t.Errorf("unexpected result: want %q, got %q", want, got)
	}

	wantStatus := http.StatusOK
	gotStatus := resp.StatusCode
	if wantStatus != gotStatus {
		t.Errorf("incorrect status code: want %d, got %d", wantStatus, gotStatus)
	}
}

func TestServe_fnReturnsInvalidMap(t *testing.T) {
	src := New().(*httpSource)
	err := config.Configure(src, map[string]interface{}{
		"address": "127.0.0.1:0",
	})
	url := fmt.Sprintf("http://%s/", src.Listener.Addr().String())

	if err != nil {
		t.Fatalf("config.Configure returned an error: %+v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		src.Serve(ctx, fn.NewFnFromInvokeFunc(func(ctx context.Context, input interface{}) (interface{}, error) {
			resp := map[string]interface{}{
				"body":       "some body",
				"statusCode": "other value",
			}

			return resp, nil
		}))
	}()

	body := "some value"
	resp, err := http.Post(url, "application/json", bytes.NewBuffer([]byte(body)))
	if err != nil {
		t.Fatalf("error posting: %+v", err)
	}

	respBody, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		t.Fatalf("ioutil.ReadAll returned error: %+v", err)
	}

	want := ""
	got := string(respBody)

	if got != want {
		t.Errorf("unexpected result: want %q, got %q", want, got)
	}

	wantStatus := http.StatusInternalServerError
	gotStatus := resp.StatusCode
	if wantStatus != gotStatus {
		t.Errorf("incorrect status code: want %d, got %d", wantStatus, gotStatus)
	}
}

func TestServe_fnProvidesResponse(t *testing.T) {
	src := New().(*httpSource)
	err := config.Configure(src, map[string]interface{}{
		"address": "127.0.0.1:0",
	})
	url := fmt.Sprintf("http://%s/", src.Listener.Addr().String())

	if err != nil {
		t.Fatalf("config.Configure returned an error: %+v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		src.Serve(ctx, fn.NewFnFromInvokeFunc(func(ctx context.Context, input interface{}) (interface{}, error) {
			resp := map[string]interface{}{
				"headers": map[string]string{
					"Content-Type": "application/custom",
				},
				"body":       "some body",
				"statusCode": http.StatusNotFound,
			}

			return resp, nil
		}))
	}()

	body := "some value"
	resp, err := http.Post(url, "application/json", bytes.NewBuffer([]byte(body)))
	if err != nil {
		t.Fatalf("error posting: %+v", err)
	}

	respBody, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		t.Fatalf("ioutil.ReadAll returned error: %+v", err)
	}

	want := "some body"
	got := string(respBody)

	if got != want {
		t.Errorf("unexpected result: want %q, got %q", want, got)
	}

	wantContentType := "application/custom"
	gotContentType := resp.Header.Get("Content-Type")
	if gotContentType != wantContentType {
		t.Errorf("incorrect content-type: want %q, got %q", wantContentType, gotContentType)
	}

	wantStatus := http.StatusNotFound
	gotStatus := resp.StatusCode
	if wantStatus != gotStatus {
		t.Errorf("incorrect status code: want %d, got %d", wantStatus, gotStatus)
	}
}

func TestServe_fnReturnsError(t *testing.T) {
	src := New().(*httpSource)
	err := config.Configure(src, map[string]interface{}{
		"address": "127.0.0.1:0",
	})
	url := fmt.Sprintf("http://%s/", src.Listener.Addr().String())

	if err != nil {
		t.Fatalf("config.Configure returned an error: %+v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		src.Serve(ctx, fn.NewFnFromInvokeFunc(func(ctx context.Context, input interface{}) (interface{}, error) {
			return nil, errors.New("some error")
		}))
	}()

	body := "some value"
	resp, err := http.Post(url, "application/json", bytes.NewBuffer([]byte(body)))
	if err != nil {
		t.Fatalf("error posting: %+v", err)
	}

	respBody, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		t.Fatalf("ioutil.ReadAll returned error: %+v", err)
	}

	want := "some error"
	got := string(respBody)

	if got != want {
		t.Errorf("unexpected result: want %q, got %q", want, got)
	}

	wantStatus := http.StatusInternalServerError
	gotStatus := resp.StatusCode
	if wantStatus != gotStatus {
		t.Errorf("incorrect status code: want %d, got %d", wantStatus, gotStatus)
	}
}

func TestServe_nonMapOutput(t *testing.T) {
	src := New().(*httpSource)
	err := config.Configure(src, map[string]interface{}{
		"address": "127.0.0.1:0",
	})
	url := fmt.Sprintf("http://%s/", src.Listener.Addr().String())

	if err != nil {
		t.Fatalf("config.Configure returned an error: %+v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		src.Serve(ctx, fn.NewFnFromInvokeFunc(func(ctx context.Context, input interface{}) (interface{}, error) {
			return 3, nil
		}))
	}()

	body := "some value"
	resp, err := http.Post(url, "application/json", bytes.NewBuffer([]byte(body)))
	if err != nil {
		t.Fatalf("error posting: %+v", err)
	}

	respBody, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		t.Fatalf("ioutil.ReadAll returned error: %+v", err)
	}

	want := ""
	got := string(respBody)

	if got != want {
		t.Errorf("unexpected result: want %q, got %q", want, got)
	}

	wantStatus := http.StatusInternalServerError
	gotStatus := resp.StatusCode
	if wantStatus != gotStatus {
		t.Errorf("incorrect status code: want %d, got %d", wantStatus, gotStatus)
	}
}

func TestServe_base64Encode(t *testing.T) {
	src := New().(*httpSource)
	err := config.Configure(src, map[string]interface{}{
		"address":           "127.0.0.1:0",
		"base64EncodeBody":  true,
		"treatOutputAsBody": true,
	})
	if err != nil {
		t.Fatalf("config.Configure returned an error: %+v", err)
	}

	url := fmt.Sprintf("http://%s/", src.Listener.Addr().String())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		src.Serve(ctx, fn.NewFnFromInvokeFunc(func(ctx context.Context, input interface{}) (interface{}, error) {
			return input.(map[string]interface{})["body"], nil
		}))
	}()

	body := "hello world"
	resp, err := http.Post(url, "application/json", bytes.NewBuffer([]byte(body)))
	if err != nil {
		t.Fatalf("error posting: %+v", err)
	}

	respBody, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		t.Fatalf("ioutil.ReadAll returned error: %+v", err)
	}

	want := "aGVsbG8gd29ybGQ="
	got := string(respBody)

	if got != want {
		t.Errorf("unexpected result: want %q, got %q", want, got)
	}

	wantStatus := http.StatusOK
	gotStatus := resp.StatusCode
	if wantStatus != gotStatus {
		t.Errorf("incorrect status code: want %d, got %d", wantStatus, gotStatus)
	}
}

func TestServe_gracefulShutdown(t *testing.T) {
	src := New().(*httpSource)
	err := config.Configure(src, map[string]interface{}{
		"address":             "127.0.0.1:0",
		"treatOutputAsBody":   true,
		"shutdownGracePeriod": "2s",
	})
	if err != nil {
		t.Fatalf("config.Configure returned an error: %+v", err)
	}

	doneCh := make(chan interface{}, 1)

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		src.Serve(ctx, identity.New())
		doneCh <- "done"
	}()

	<-time.After(50 * time.Millisecond)
	cancel()

	select {
	case <-time.After(3 * time.Second):
		t.Error("server did not exit within the grace period")
	case <-doneCh:
		break
	}
}
