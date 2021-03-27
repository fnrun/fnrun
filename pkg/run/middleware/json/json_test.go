package json

import (
	"context"
	"errors"
	"math"
	"reflect"
	"testing"

	"github.com/fnrun/fnrun/pkg/fn"
	"github.com/fnrun/fnrun/pkg/run/config"
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

func TestTranscode_serialize(t *testing.T) {
	output, err := transcode(map[string]interface{}{"a": "some value"}, "serialize")
	if err != nil {
		t.Errorf("trancode returned error: %+v", err)
	}

	want := `{"a":"some value"}`
	got, ok := output.(string)
	if !ok {
		t.Errorf("expected output to be a string but it was a %T", output)
	}

	if got != want {
		t.Errorf("unexpected output: want %q, got %q", want, got)
	}
}

func TestTranscode_serializeWithInvalidInput(t *testing.T) {
	output, err := transcode(math.NaN(), "serialize")
	if err == nil {
		t.Error("expected transcode to return error but it did not")
	}

	if output != nil {
		t.Errorf("expected output to be nil but it was %#v", output)
	}
}

func TestTranscode_deserialize(t *testing.T) {
	output, err := transcode(`{"a": "some value"}`, "deserialize")
	if err != nil {
		t.Fatalf("transcode returned error: %+v", err)
	}

	want := map[string]interface{}{"a": "some value"}
	got := output.(map[string]interface{})
	if !reflect.DeepEqual(got, want) {
		t.Errorf("unexpected output:\nwant %#v\ngot  %#v", want, got)
	}
}

func TestTranscode_deserializedWithInvalidInput(t *testing.T) {
	output, err := transcode(3, "deserialize")
	if err == nil {
		t.Fatal("expected transcode to return error but it did not")
	}

	if output != nil {
		t.Errorf("expected output to be nil but it was %#v", output)
	}

	got := err.Error()
	want := "expected string value but received one of type int"
	if got != want {
		t.Errorf("error message: want %q, got %q", want, got)
	}
}

func TestTranscode_unknownStrategy(t *testing.T) {
	output, err := transcode(`{"a": 3}`, "unknown")
	if err != errUnknownStrategy {
		t.Errorf("unexpected error: want %+v, got %+v", errUnknownStrategy, err)
	}
	if output != nil {
		t.Errorf("expected output to be nil but was: %+v", output)
	}
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

func TestInvoke_invalidInput(t *testing.T) {
	m := New()
	err := config.Configure(m, map[string]interface{}{"input": "serialize"})
	if err != nil {
		t.Fatalf("config.Configure returned error: %+v", err)
	}

	f := fn.NewFnFromInvokeFunc(func(context.Context, interface{}) (interface{}, error) {
		return "called", nil
	})

	output, err := m.Invoke(context.Background(), math.NaN(), f)
	if err == nil {
		t.Error("expected Invoke to return an error but it did not")
	}
	if output != nil {
		t.Errorf("expected output to be nil but it was %#v", output)
	}
}

func TestInvoke_invalidOutput(t *testing.T) {
	m := New()
	err := config.Configure(m, map[string]interface{}{"output": "serialize"})
	if err != nil {
		t.Fatalf("config.Configure returned error: %+v", err)
	}

	f := fn.NewFnFromInvokeFunc(func(context.Context, interface{}) (interface{}, error) {
		return math.NaN(), nil
	})

	output, err := m.Invoke(context.Background(), "some input", f)
	if err == nil {
		t.Error("expected Invoke to return an error but it did not")
	}
	if output != nil {
		t.Errorf("expected output to be nil but it was %#v", output)
	}
}

func TestInvoke_whenFunctionReturnsErrorDoesNotTranscodeOutput(t *testing.T) {
	m := New()
	err := config.Configure(m, map[string]interface{}{"output": "deserialize"})
	if err != nil {
		t.Fatalf("config.Configure returned error: %+v", err)
	}

	f := fn.NewFnFromInvokeFunc(func(context.Context, interface{}) (interface{}, error) {
		return `{"a": "some string"}`, errors.New("some error message")
	})

	output, err := m.Invoke(context.Background(), "input value", f)
	if err == nil {
		t.Fatal("expected Invoke to return error but it did not")
	}
	if output != nil {
		t.Errorf("expected output to be nil but it was %#v", output)
	}

	want := "some error message"
	got := err.Error()
	if got != want {
		t.Errorf("unexpected error message: want %q, got %q", want, got)
	}
}

func TestInvoke_inputDeserialization(t *testing.T) {
	f, c := createReceiverFn()
	m := New()
	err := config.Configure(m, map[string]interface{}{"input": "deserialize"})
	if err != nil {
		t.Fatalf("config.Configure returned an error: %+v", err)
	}

	if _, err = m.Invoke(context.Background(), `{"a": "b"}`, f); err != nil {
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
