package json

import (
	"context"
	"reflect"
	"testing"

	"github.com/fnrun/fnrun/pkg/fn"
)

// createReceiverFn creates and returns an Fn and channel. When invoked,
// the function will publish its input to the channel, and then return it
// without error.
func createReceiverFn() (fn.Fn, chan interface{}) {
	receiveChan := make(chan interface{}, 10)

	f := fn.NewFnFromInvokeFunc(func(ctx context.Context, input interface{}) (interface{}, error) {
		receiveChan <- input
		return input, nil
	})

	return f, receiveChan
}

func TestInvoke(t *testing.T) {
	f, _ := createReceiverFn()

	m := &jsonMiddleware{
		config: &jsonMiddlewareConfig{
			Input:  "deserialize",
			Output: "serialize",
		},
	}

	output, err := m.Invoke(context.Background(), `{"a": "b"}`, f)
	if err != nil {
		t.Fatalf("Invoke returned error: %#v", err)
	}

	want := `{"a":"b"}`
	got, ok := output.(string)
	if !ok {
		t.Fatalf("output was not string but was %T", output)
	}

	if got != want {
		t.Errorf("want %q; got %q", want, got)
	}
}

func TestInvoke_inputDeserialization(t *testing.T) {
	f, c := createReceiverFn()

	m := &jsonMiddleware{
		config: &jsonMiddlewareConfig{Input: "deserialize"},
	}

	if _, err := m.Invoke(context.Background(), `{"a": "b"}`, f); err != nil {
		t.Fatalf("Invoke returned error: %#v", err)
	}

	want := map[string]interface{}{
		"a": "b",
	}
	got := <-c

	if !reflect.DeepEqual(got, want) {
		t.Errorf("Invoke did not receive expected input: want %#v, got %#v", want, got)
	}
}

func TestInvoke_inputSerialization(t *testing.T) {
	f, c := createReceiverFn()

	m := &jsonMiddleware{
		config: &jsonMiddlewareConfig{Input: "serialize"},
	}

	input := map[string]interface{}{"a": "b"}

	if _, err := m.Invoke(context.Background(), input, f); err != nil {
		t.Fatalf("Invoke returned error: %#v", err)
	}

	received := <-c

	want := `{"a":"b"}`
	got, ok := received.(string)
	if !ok {
		t.Fatalf("received value was not string but was %T", received)
	}

	if got != want {
		t.Errorf("Invoke did not receive expected input: want %#v, got %#v", want, got)
	}
}

func TestInvoke_outputSerialization(t *testing.T) {
	f, _ := createReceiverFn()

	m := &jsonMiddleware{
		config: &jsonMiddlewareConfig{Output: "serialize"},
	}

	input := map[string]interface{}{"a": "b"}

	output, err := m.Invoke(context.Background(), input, f)
	if err != nil {
		t.Fatalf("Invoke returned error: %#v", err)
	}

	want := `{"a":"b"}`
	got, ok := output.(string)
	if !ok {
		t.Fatalf("output was not string but was %T", output)
	}

	if got != want {
		t.Errorf("want %q; got %q", want, got)
	}
}

func TestInvoke_outputDeserialization(t *testing.T) {
	f, _ := createReceiverFn()

	m := &jsonMiddleware{
		config: &jsonMiddlewareConfig{Output: "deserialize"},
	}

	got, err := m.Invoke(context.Background(), `{"a":"b"}`, f)
	if err != nil {
		t.Fatalf("Invoke returned error: %#v", err)
	}

	want := map[string]interface{}{"a": "b"}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("Invoke did not return expected output: want %#v, got %#v", want, got)
	}
}

func TestNew_defaultValues(t *testing.T) {
	m := New().(*jsonMiddleware)

	gotInputStrategy := m.config.Input
	wantInputStrategy := ""

	if gotInputStrategy != wantInputStrategy {
		t.Errorf("unexpected input strategy, want: %q, got %q", wantInputStrategy, gotInputStrategy)
	}

	gotOutputStrategy := m.config.Output
	wantOutputStrategy := ""

	if gotOutputStrategy != wantOutputStrategy {
		t.Errorf("unexpected output strategy, want: %q, got %q", wantOutputStrategy, gotOutputStrategy)
	}
}

func TestConfigureMap(t *testing.T) {
	m := New().(*jsonMiddleware)
	m.ConfigureMap(map[string]interface{}{
		"input":  "serialize",
		"output": "deserialize",
	})

	gotInputStrategy := m.config.Input
	wantInputStrategy := "serialize"

	if gotInputStrategy != wantInputStrategy {
		t.Errorf("unexpected input strategy, want: %q, got %q", wantInputStrategy, gotInputStrategy)
	}

	gotOutputStrategy := m.config.Output
	wantOutputStrategy := "deserialize"

	if gotOutputStrategy != wantOutputStrategy {
		t.Errorf("unexpected output strategy, want: %q, got %q", wantOutputStrategy, gotOutputStrategy)
	}
}
