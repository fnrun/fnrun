package fn

import "context"

// Fn represents a function that encapsulates some application functionality.
//
// Fn implementations may contain the functionality directly or facilitate some
// process by which the functionality is invoked.
type Fn interface {
	Invoke(context.Context, interface{}) (interface{}, error)
}
