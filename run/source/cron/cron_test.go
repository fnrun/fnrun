package cron

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/fnrun/fnrun/fn"
	"github.com/fnrun/fnrun/run/config"
)

func nullInvokeFunc(context.Context, interface{}) (interface{}, error) {
	return nil, nil
}

func TestInvoke(t *testing.T) {
	wg := &sync.WaitGroup{}
	wg.Add(1)
	done := false

	f := fn.NewFnFromInvokeFunc(func(context.Context, interface{}) (interface{}, error) {
		if done {
			return nil, nil
		}

		done = true
		wg.Done()
		return nil, nil
	})

	m := New()
	if err := config.Configure(m, "* * * * * *"); err != nil {
		t.Fatalf("Configuring middleware returned error: %#v", err)
	}

	go func() {
		m.Serve(context.Background(), f)
	}()

	select {
	case <-time.After(2 * time.Second):
		t.Error("job did not run within two seconds")
	case <-wait(wg):
		break
	}
}

func TestInvoke_withoutConfiguring(t *testing.T) {
	m := New()
	err := m.Serve(context.Background(), fn.NewFnFromInvokeFunc(nullInvokeFunc))

	if err == nil {
		t.Error("expected Serve to return an error but it did not")
	}
}

func TestRequiresConfig(t *testing.T) {
	m := New()
	err := config.Configure(m, nil)

	if err == nil {
		t.Error("expected config.Configure to return an error but it did not")
	}
}

func TestConfigureString_invalidCronspec(t *testing.T) {
	m := New().(*cronSource)
	err := m.ConfigureString("some invalid string")

	if err == nil {
		t.Error("expected ConfigureString to return an error but it did not")
	}
}

func wait(wg *sync.WaitGroup) chan bool {
	ch := make(chan bool)
	go func() {
		wg.Wait()
		ch <- true
	}()
	return ch
}
