// Package healthcheck provides a simple HTTP health check. It sets up an HTTP
// server on port 8080 that returns a 200 response from the `/` route.
package healthcheck

import (
	"context"
	"net/http"

	"github.com/fnrun/fnrun/fn"
	"github.com/fnrun/fnrun/run"
)

type healthcheckMiddleware struct {
}

func (h *healthcheckMiddleware) Configure() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	go func() {
		http.ListenAndServe(":8080", mux)
	}()

	return nil
}

func (h *healthcheckMiddleware) Invoke(ctx context.Context, input interface{}, f fn.Fn) (interface{}, error) {
	return f.Invoke(ctx, input)
}

// New returns a new healthcheck middleware.
func New() run.Middleware {
	return &healthcheckMiddleware{}
}
