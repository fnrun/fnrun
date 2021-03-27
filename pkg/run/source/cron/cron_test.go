package cron

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/fnrun/fnrun/pkg/fn"
	"github.com/fnrun/fnrun/pkg/run/config"
)

func TestInvoke(t *testing.T) {
	wg := &sync.WaitGroup{}

	f := fn.NewFnFromInvokeFunc(func(context.Context, interface{}) (interface{}, error) {
		wg.Done()
		return nil, nil
	})

	m := New()
	config.Configure(m, "1 * * * * *")

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

func wait(wg *sync.WaitGroup) chan bool {
	ch := make(chan bool)
	go func() {
		wg.Wait()
		ch <- true
	}()
	return ch
}
