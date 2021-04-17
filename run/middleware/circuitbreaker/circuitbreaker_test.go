package circuitbreaker

import (
	"context"
	"errors"
	"testing"

	"github.com/sony/gobreaker"
)

func TestCircuitBreakerMiddleware_Invoke_tripsAfterConfiguredFailures(t *testing.T) {
	m := New().(*circuitBreakerMiddleware)

	m.ConfigureMap(map[string]interface{}{
		"maxRequests": 2,
	})

	f := &requestCounterFn{}

	for i := 0; i < 6; i++ {
		_, err := m.Invoke(context.Background(), "some input", f)
		if err != errExample {
			t.Errorf("Pass %d: want %+v, got %+v", i, errExample, err)
		}
	}

	_, err := m.Invoke(context.Background(), "some input", f)
	if err != gobreaker.ErrOpenState {
		t.Errorf("Wrong error: want %+v, got %+v", gobreaker.ErrOpenState, err)
	}
}

// -----------------------------------------------------------------------------
// Test fn

var errExample = errors.New("some error occurred")

type requestCounterFn struct {
	RequestCount uint32
}

func (rcf *requestCounterFn) Invoke(context.Context, interface{}) (interface{}, error) {
	rcf.RequestCount++
	return nil, errExample
}
