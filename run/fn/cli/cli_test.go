package cli

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/fnrun/fnrun/run/config"
)

func TestNew_withoutConfigReturnsUnconfiguredError(t *testing.T) {
	f := New()
	_, err := f.Invoke(context.Background(), "some input")
	if err != ErrUnconfiguredCmd {
		t.Fatalf("expected Invoke to return ErrUnconfiguredCmd but returned: %+v", err)
	}
}

func TestNew_withStringConfiguration(t *testing.T) {
	f := New().(*cliFn)
	err := f.ConfigureString("./myprogram")
	if err != nil {
		t.Fatalf("ConfigureString returned error: %+v", err)
	}

	_, ok := f.f.(*service)
	if !ok {
		t.Errorf("expected fn to be a *service but was a %T", f.f)
	}
}

func TestNew_withServiceMapConfiguration(t *testing.T) {
	configMap := map[string]interface{}{
		"command": fmt.Sprintf("%s -test.run=%s", os.Args[0], exeName("Test_HelperSubprocess")),
		"env":     []string{"GO_RUNNING_SUBPROCESS=1"},
	}
	f := New().(*cliFn)
	err := config.Configure(f, configMap)
	if err != nil {
		t.Fatalf("Configure returned err: %+v", err)
	}

	_, ok := f.f.(*service)
	if !ok {
		t.Errorf("expected fn to be a *service but was %T", f.f)
	}
}

func TestNew_withScriptConfiguration(t *testing.T) {
	configMap := map[string]interface{}{
		"command": fmt.Sprintf("%s -test.run=%s", os.Args[0], exeName("Test_HelperSubprocess")),
		"env":     []string{"GO_RUNNING_SUBPROCESS=1"},
		"script":  true,
	}
	f := New().(*cliFn)
	err := config.Configure(f, configMap)
	if err != nil {
		t.Fatalf("Configure returned err: %+v", err)
	}

	_, ok := f.f.(*script)
	if !ok {
		t.Errorf("expected fn to be a *script but was %T", f.f)
	}
}

func TestNew_withNilConfig(t *testing.T) {
	f := New()
	err := config.Configure(f, nil)
	if err == nil {
		t.Errorf("expected Configure to return an error but was nil")
	}
}

func TestCliFn_Invoke(t *testing.T) {
	configMap := map[string]interface{}{
		"command": fmt.Sprintf("%s -test.run=%s", os.Args[0], exeName("Test_HelperSubprocess")),
		"env":     []string{"GO_RUNNING_SUBPROCESS=1"},
	}
	f := New()
	err := config.Configure(f, configMap)
	if err != nil {
		t.Fatalf("Configure returned err: %+v", err)
	}

	output, err := f.Invoke(context.Background(), "some input")
	if err != nil {
		t.Fatalf("Invoke returned error: %+v", err)
	}

	want := "from subprocess: some input"
	got := output.(string)

	if got != want {
		t.Errorf("unexpected output: want %q, got %q", want, got)
	}
}
