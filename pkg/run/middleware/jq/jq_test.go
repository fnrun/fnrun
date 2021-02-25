package jq

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/fnrun/fnrun/pkg/fn"
	"github.com/fnrun/fnrun/pkg/run/fn/identity"
)

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
