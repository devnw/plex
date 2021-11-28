package plex

import (
	"context"
	"fmt"
)

// _ctx returns a valid context with cancel even if it the
// supplied context is initially nil. If the supplied context
// is nil it uses the background context
func _ctx(c context.Context) (context.Context, context.CancelFunc) {
	if c == nil {
		c = context.Background()
	}

	return context.WithCancel(c)
}

func merge(parent, child context.Context) context.Context {
	if child == nil {
		return parent
	}

	return child
}

type errWrapper struct {
	err        string
	underlying error
}

func (e *errWrapper) Error() string {
	return e.err
}

func (e *errWrapper) Unwrap() error {
	return e.underlying
}

func recoverErr(err error, r interface{}) error {
	switch v := r.(type) {
	case nil:
		if err != nil {
			return err
		}

		return nil
	case string:
		return &errWrapper{
			err:        v,
			underlying: err,
		}
	case error:
		return &errWrapper{
			err:        v.Error(),
			underlying: err,
		}
	default:
		// Fallback err (per specs, error strings
		// should be lowercase w/o punctuation
		return &errWrapper{
			err:        fmt.Sprintf("panic: %v", r),
			underlying: err,
		}
	}
}
