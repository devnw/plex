package plex

import "errors"

// ErrTimeout is returned when a request times out.
var ErrTimeout = errors.New("timeout")

// ErrClosed is returned when an internal channel in the multiplexer is closed.
var ErrClosed = errors.New("closed")

// ErrNil is returned when an unhandled nil value is passed to a function.
var ErrNil = errors.New("nil value")
