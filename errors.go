package plex

import "errors"

// ErrTimeout is returned when a request times out.
var ErrTimeout = errors.New("timeout")

// ErrClosed is returned when an internal channel in the multiplexer is closed.
var ErrClosed = errors.New("closed")

// ErrNil is returned when an unhandled nil value is passed to a function.
var ErrNil = errors.New("nil value")

// ErrEmptyPool when no io.Readers / io.Writers / io.ReadWriters
// are passed to the multiplexer during creation.
var ErrEmptyPool = errors.New("empty pools")

// TODO: Add error for when a timeout is exceeded on a
// request for a reader or writer from the pool (e.g. when
// a reader is requested but no readers are available).
// This should be configurable through the options, or
// the consumer can choose to wait for the next available
// reader or writer.
