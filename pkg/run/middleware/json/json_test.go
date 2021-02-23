package json

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/fnrun/fnrun/pkg/run/fn/identity"
)

func TestJsonMiddleware_Invoke(t *testing.T) {
	f := identity.New()
	ctx := context.Background()
	m := New()
	input := map[string]interface{}{
		"a": "b",
	}

	output, err := m.Invoke(ctx, input, f)
	if err != nil {
		t.Errorf("Invoke returned err: %#v", err)
	}

	want := input
	got := output

	if !reflect.DeepEqual(got, want) {
		t.Errorf("Invoke output: want %#v, got %#v", want, got)
	}
}

func TestJsonMiddleware_Invoke_serializeInputOnly(t *testing.T) {
	f := identity.New()
	ctx := context.Background()
	m := &jsonMiddleware{
		config: &jsonMiddlewareConfig{
			serializeInput:    true,
			deserializeOutput: false,
		},
	}
	input := map[string]interface{}{
		"a": "b",
	}

	output, err := m.Invoke(ctx, input, f)
	if err != nil {
		t.Errorf("Invoke returned err: %#v", err)
	}

	want := `{"a":"b"}`
	got := output

	if got != want {
		t.Errorf("Invoke output: want %q, got %q", want, got)
	}
}

func TestJsonMiddleware_Invoke_deserializeOutputOnly(t *testing.T) {
	f := &jsonInputWrapperFn{}
	ctx := context.Background()
	m := &jsonMiddleware{
		config: &jsonMiddlewareConfig{
			serializeInput:    false,
			deserializeOutput: true,
		},
	}
	input := "some string value"

	output, err := m.Invoke(ctx, input, f)
	if err != nil {
		t.Errorf("Invoke returned err: %#v", err)
	}

	want := map[string]interface{}{
		"input": input,
	}
	got := output

	if !reflect.DeepEqual(got, want) {
		t.Errorf("Invoke output: want %#v, got %#v", want, got)
	}
}

func TestJsonMiddleware_ConfigString(t *testing.T) {
	t.Run("No config", func(t *testing.T) {
		m := New().(*jsonMiddleware)

		if m.config.serializeInput != true {
			t.Error("expected input serialization to be enabled")
		}
		if m.config.deserializeOutput != true {
			t.Error("expected output deserialization to be enabled")
		}
	})

	t.Run(`Configure with "input" string`, func(t *testing.T) {
		m := New().(*jsonMiddleware)
		if err := m.ConfigureString("input"); err != nil {
			t.Fatal(err)
		}

		if m.config.serializeInput != true {
			t.Error("expected input serialization to be enabled")
		}
		if m.config.deserializeOutput != false {
			t.Error("expected output deserialization to be disabled")
		}
	})

	t.Run(`Configure with "output" string`, func(t *testing.T) {
		m := New().(*jsonMiddleware)
		if err := m.ConfigureString("output"); err != nil {
			t.Fatal(err)
		}

		if m.config.serializeInput != false {
			t.Error("expected input serialization to be disabled")
		}
		if m.config.deserializeOutput != true {
			t.Error("expected output deserialization to be enabled")
		}
	})

	t.Run(`Configure with "both" string`, func(t *testing.T) {
		m := New().(*jsonMiddleware)
		if err := m.ConfigureString("both"); err != nil {
			t.Fatal(err)
		}

		if m.config.serializeInput != true {
			t.Error("expected input serialization to be enabled")
		}
		if m.config.deserializeOutput != true {
			t.Error("expected output deserialization to be enabled")
		}
	})

	t.Run("Configure with unsupported string", func(t *testing.T) {
		m := New().(*jsonMiddleware)
		err := m.ConfigureString("some unsupported value")

		if err == nil {
			t.Fatal("Expected err to have a value")
		}
		got := err.Error()
		want := `unsupported config value: "some unsupported value"; expected "input", "output", or "both"`

		if got != want {
			t.Errorf("ConfigureString did not return correct err: want %q, got %q", want, got)
		}

		if m.config.serializeInput != true {
			t.Error("expected input serialization to be enabled")
		}
		if m.config.deserializeOutput != true {
			t.Error("expected output deserialization to be enabled")
		}
	})
}

// -----------------------------------------------------------------------------
// Example fns

type jsonInputWrapperFn struct{}

func (*jsonInputWrapperFn) Invoke(ctx context.Context, input interface{}) (interface{}, error) {
	output := map[string]interface{}{
		"input": input,
	}

	jsonBytes, err := json.Marshal(output)
	if err != nil {
		return nil, err
	}

	return string(jsonBytes), nil
}
