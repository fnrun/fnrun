package fn

import (
	"context"
	"errors"
	"fmt"
	"testing"
)

func TestNewFnFromInvokeFunc(t *testing.T) {
	f := NewFnFromInvokeFunc(func(ctx context.Context, input interface{}) (interface{}, error) {
		s, ok := input.(string)
		if !ok {
			return nil, errors.New("could not convert input to string")
		}

		output := fmt.Sprintf("Hello, %s!", s)
		return output, nil
	})

	output, err := f.Invoke(context.Background(), "world")
	if err != nil {
		t.Fatalf("Invoke returned error: %#v", err)
	}

	want := "Hello, world!"
	got := output.(string)

	if got != want {
		t.Errorf("Outputs did not match: want %q, got %q", want, got)
	}
}
