// Package key provides a middleware that selects a single key from an input and
// uses it as the input for the Fn. The input must be of type
// map[string]interface{}, otherwise the middleware returns an error and does
// not invoke the Fn.
//
// The middleware may be configured with a string containing the name of the key
// to extract from input, and it defaults to the empty string.
package key

import (
	"context"
	"fmt"

	"github.com/fnrun/fnrun/fn"
	"github.com/fnrun/fnrun/run"
)

type keyMiddleware struct {
	Key string
}

func (k *keyMiddleware) ConfigureString(key string) error {
	k.Key = key
	return nil
}

func (k *keyMiddleware) Invoke(ctx context.Context, input interface{}, f fn.Fn) (interface{}, error) {
	m, ok := input.(map[string]interface{})
	if !ok {
		err := fmt.Errorf("key middleware expected input to be of type map[string]interface{}, but it is %T", input)
		return nil, err
	}

	return f.Invoke(ctx, m[k.Key])
}

// New returns a key middleware configured with the key as an empty string.
func New() run.Middleware {
	return &keyMiddleware{}
}

// NewWithKey returns a key middleware configured with key.
func NewWithKey(key string) run.Middleware {
	return &keyMiddleware{Key: key}
}
