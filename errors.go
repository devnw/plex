package plex

import (
	"errors"
	"fmt"
	"net"
)

// errTimeout is returned when a request times out.
var errTimeout = errors.New("timeout")

// errClosed is returned when an internal channel in the multiplexer is closed.
var errClosed = errors.New("closed")

var errConnectorNil = errors.New("connector method is nil")

var errInvalidMaxCapacity = errors.New("max capacity must be greater than zero")

var errTooManyConns = errors.New("connection count exceeds configured capacity")

var errInvalidTimeout = errors.New("timeout must be greater than zero")

var errImproperAutoScalingNilConnector = errors.New(
	"auto-scaling connector cannot be nil",
)

type ErrConnection struct {
	net.Addr
	error
}

func disconnected(addr net.Addr) error {
	return &ErrConnection{
		addr,
		errors.New("disconnected"),
	}
}

type errAddrMismatch struct {
	Expected net.Addr
	Actual   net.Addr
}

func (e errAddrMismatch) Error() string {
	return fmt.Sprintf(
		"address mismatch; expected: %s:%s; actual: %s:%s",
		e.Expected.Network(),
		e.Expected,
		e.Actual.Network(),
		e.Actual,
	)
}
