package jq

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/fnrun/fnrun/pkg/fn"
	"github.com/fnrun/fnrun/pkg/run/config"
	"github.com/fnrun/fnrun/pkg/run/fn/identity"
)

func TestConfigure_invalidConfig(t *testing.T) {
	m := New()
	err := config.Configure(m, nil)

	if err == nil {
		t.Error("expected config.Configure to return an error but it did not")
	}
}

func TestConfigureMap_invalidDestructure(t *testing.T) {
	m := New()
	err := config.Configure(m, map[string]interface{}{"input": 4})
	if err == nil {
		t.Fatal("expected config.Configure to return an error but it did not")
	}

	want := "1 error(s) decoding:\n\n* 'input' expected type 'string', got unconvertible type 'int', value: '4'"
	got := err.Error()

	if got != want {
		t.Errorf("unexpected error message: want %q, got %q", want, got)
	}
}

func TestConfigureMap_invalidInputConfiguration(t *testing.T) {
	m := New()
	err := config.Configure(m, map[string]interface{}{"input": "["})
	if err == nil {
		t.Error("expected config.Configure to return an error but it did not")
	}

	want := "invalid input configuration: unexpected token <EOF>"
	got := err.Error()

	if got != want {
		t.Errorf("unexpected error message: want %q, got %q", want, got)
	}
}

func TestConfigureMap_invalidOutputConfiguration(t *testing.T) {
	m := New()
	err := config.Configure(m, map[string]interface{}{"output": "["})
	if err == nil {
		t.Error("expected config.Configure to return an error but it did not")
	}

	want := "invalid output configuration: unexpected token <EOF>"
	got := err.Error()

	if got != want {
		t.Errorf("unexpected error message: want %q, got %q", want, got)
	}
}

func TestInvoke_inputOnly_singleOutput(t *testing.T) {
	f := identity.New()

	m := New().(*jqMiddleware)
	err := m.ConfigureMap(map[string]interface{}{
		"input": ".a.b",
	})
	if err != nil {
		t.Fatalf("ConfigMap returned error: %#v", err)
	}

	var input map[string]interface{}
	if err := json.Unmarshal([]byte(`{"a": {"b": 42}}`), &input); err != nil {
		t.Fatalf("Unmarshaling JSON gave error: %#v", err)
	}

	output, err := m.Invoke(context.Background(), input, f)
	if err != nil {
		t.Errorf("Invoke returned error: %#v", err)
	}

	want := float64(42)
	got := output
	if got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
}

func TestInvoke_inputOnly(t *testing.T) {
	f := identity.New()

	m := New().(*jqMiddleware)
	err := m.ConfigureMap(map[string]interface{}{
		"input": ".[1,2]",
	})
	if err != nil {
		t.Fatalf("ConfigMap returned error: %#v", err)
	}

	input := []interface{}{"a", "b", "c", "d"}

	output, err := m.Invoke(context.Background(), input, f)
	if err != nil {
		t.Errorf("Invoke returned error: %#v", err)
	}

	want := []interface{}{"b", "c"}
	got := output

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got: %v, want: %v", got, want)
	}
}

func TestInvoke(t *testing.T) {
	f := fn.NewFnFromInvokeFunc(func(ctx context.Context, input interface{}) (interface{}, error) {
		result := fmt.Sprint(input) + "Result"
		return map[string]interface{}{
			"someValue": result,
		}, nil
	})

	m := New().(*jqMiddleware)
	err := m.ConfigureMap(map[string]interface{}{
		"input":  ".[0]",
		"output": ".someValue",
	})
	if err != nil {
		t.Fatalf("ConfigMap returned error: %#v", err)
	}

	input := []interface{}{"a", "b", "c", "d"}

	output, err := m.Invoke(context.Background(), input, f)
	if err != nil {
		t.Errorf("Invoke returned error: %#v", err)
	}

	want := "aResult"
	got := output

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got: %v, want: %v", got, want)
	}
}

func TestInvoke_unsatsifiableInputQuery(t *testing.T) {
	m := New()
	err := config.Configure(m, map[string]interface{}{"input": ".x"})
	if err != nil {
		t.Fatalf("config.Configure returned error: %+v", err)
	}

	output, err := m.Invoke(context.Background(), "invalid input", identity.New())

	if err == nil {
		t.Error("expected Invoke to return an error but it did not")
	}

	if output != nil {
		t.Errorf("expected output to be nil but it was %#v", output)
	}
}

func TestInvoke_whenFnReturnsErrorDoesNotProcessOutput(t *testing.T) {
	m := New()
	err := config.Configure(m, map[string]interface{}{"output": ".x"})
	if err != nil {
		t.Fatalf("config.Configure returned an error: %+v", err)
	}
	errCustom := errors.New("custom error")
	f := fn.NewFnFromInvokeFunc(func(context.Context, interface{}) (interface{}, error) {
		return "some value", errCustom
	})

	output, err := m.Invoke(context.Background(), "some input", f)
	if err != errCustom {
		t.Errorf("unexpected error from Invoke: want %+v, got %+v", errCustom, err)
	}

	wantOutput := "some value"
	gotOutput, ok := output.(string)
	if !ok {
		t.Errorf("expected output to be a string but it was a %T", output)
	}

	if gotOutput != wantOutput {
		t.Errorf("unexpected output: want %q, got %q", wantOutput, gotOutput)
	}
}
