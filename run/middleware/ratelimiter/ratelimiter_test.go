package ratelimiter

import (
	"context"
	"testing"
	"time"

	"github.com/fnrun/fnrun/run/config"
	"github.com/fnrun/fnrun/run/fn/identity"
)

func TestConfigureMap(t *testing.T) {
	m := New().(*rlmiddleware)

	burst := 5
	everyStr := "10s"
	every, _ := time.ParseDuration(everyStr)

	err := config.Configure(m, map[string]interface{}{
		"burst": burst,
		"every": everyStr,
	})
	if err != nil {
		t.Fatalf("config.Configure returned an error: %+v", err)
	}

	if m.Burst != burst {
		t.Errorf("burst values did not match: want %d, got %d", burst, m.Burst)
	}
	if m.Every != every {
		t.Errorf("every values did not match: want %v, got %v", every, m.Every)
	}
}

func TestConfigureMap_defaultEvery(t *testing.T) {
	m := New().(*rlmiddleware)

	burst := 5
	every := 1 * time.Second

	err := config.Configure(m, map[string]interface{}{
		"burst": burst,
	})
	if err != nil {
		t.Fatalf("config.Configure returned an error: %+v", err)
	}

	if m.Burst != burst {
		t.Errorf("burst values did not match: want %d, got %d", burst, m.Burst)
	}
	if m.Every != every {
		t.Errorf("every values did not match: want %v, got %v", every, m.Every)
	}
}

func TestInvoke(t *testing.T) {
	m := New()
	f := identity.New()
	ctx := context.Background()

	output, err := m.Invoke(ctx, "some input", f)
	if err != nil {
		t.Errorf("Invoke returned error: %+v", err)
	}

	want := "some input"
	got, ok := output.(string)
	if !ok {
		t.Fatalf("expected output to be a string but it was a %T", output)
	}
	if got != want {
		t.Errorf("outputs did not match: want %q, got %q", want, got)
	}
}

func TestInvoke_cancelledContext(t *testing.T) {
	m := New()
	f := identity.New()
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel now to make wait return error

	output, err := m.Invoke(ctx, "some value", f)
	if err == nil {
		t.Errorf("expected Invoke to return an error but it did not")
	}

	if output != nil {
		t.Errorf("expected output to be nil but it was %#v", output)
	}
}
