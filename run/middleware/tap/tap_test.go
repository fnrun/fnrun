package tap

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/fnrun/fnrun/run/config"
)

type testFn struct {
	WillError bool
}

func (f *testFn) Invoke(ctx context.Context, input interface{}) (interface{}, error) {
	if f.WillError {
		return nil, errors.New("some error message")
	}
	return input, nil
}

func TestNew_withoutConfigReturnsUnconfiguredError(t *testing.T) {
	m := New()
	f := &testFn{}
	_, err := m.Invoke(context.Background(), "some input", f)
	if err != ErrUnconfiguredCmd {
		t.Fatalf("expected Invoke to return ErrUnconfiguredCmd but returned: %+v", err)
	}
}

func TestNew_withStringConfiguration(t *testing.T) {
	m := New().(*tapMiddleware)
	err := config.Configure(m, "./myprogram")
	if err != nil {
		t.Fatalf("ConfigureString returned error: %+v", err)
	}

	if m.baseCmd == nil {
		t.Fatalf("the baseCmd was not set on the middleware")
	}
}

func TestNew_withMapConfiguration(t *testing.T) {
	configMap := map[string]interface{}{
		"command":   fmt.Sprintf("%s -test.run=%s", os.Args[0], "Test_HelperSubprocess"),
		"env":       []string{"GO_RUNNING_SUBPROCESS=1"},
		"tapInput":  false,
		"tapOutput": false,
		"tapError":  false,
	}
	m := New().(*tapMiddleware)
	err := config.Configure(m, configMap)
	if err != nil {
		t.Fatalf("Configure returned an error: %+v", err)
	}

	if m.baseCmd == nil {
		t.Error("the baseCmd was not set on the middleware")
	}
	if m.tapInput {
		t.Error("tapInput was not disabled")
	}
	if m.tapOutput {
		t.Error("tapOutput was not disabled")
	}
	if m.tapError {
		t.Error("tapError was not disabled")
	}
}

func TestNew_withNilConfig(t *testing.T) {
	m := New()
	err := config.Configure(m, nil)
	if err == nil {
		t.Errorf("expected Configure to return an error but was nil")
	}
}
