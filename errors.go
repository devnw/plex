package plex

import "errors"

var ErrTimeout = errors.New("timeout")
var ErrClosed = errors.New("closed")
var ErrNil = errors.New("nil value")
