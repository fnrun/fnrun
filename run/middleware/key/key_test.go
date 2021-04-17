package key

import (
	"context"
	"testing"
)

func input() map[string]interface{} {
	return map[string]interface{}{
		"body": "some body value",
		"headers": map[string]string{
			"Content-Type":    "text/plain",
			"X-Custom-Header": "some value",
		},
	}
}

func TestInvoke(t *testing.T) {
	m := NewWithKey("body")
	f := &echoFn{}

	output, err := m.Invoke(context.Background(), input(), f)
	if err != nil {
		t.Fatalf("Invoke returned error: %#v", err)
	}

	want := "some body value"
	got := output.(string)

	if got != want {
		t.Errorf("Invoke: want %q; got %q", want, got)
	}
}

func TestInvoke_typeMismatch(t *testing.T) {
	m := NewWithKey("body")
	f := &echoFn{}

	ctx := context.Background()
	badInput := "some string"

	output, err := m.Invoke(ctx, badInput, f)
	if err == nil {
		t.Error("expected Invoke to return an error due to type mismatch")
	}

	if output != nil {
		t.Errorf("expected output to be nil but it was %#v", output)
	}
}

func TestConfigureString(t *testing.T) {
	m := New()
	k := m.(*keyMiddleware)

	k.ConfigureString("someKey")

	want := "someKey"
	got := k.Key

	if got != want {
		t.Errorf("setting key failed: want %q, got %q", want, got)
	}
}

// -----------------------------------------------------------------------------
// Sample fn

type echoFn struct{}

func (*echoFn) Invoke(_ context.Context, input interface{}) (interface{}, error) {
	return input, nil
}
