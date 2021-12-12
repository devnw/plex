package plex

import (
	"net"
	"time"
)

// WithMaxCapacity allocates the internal buffer for the multiplexer to a
// static capacity not based on the number of connections supplied at
// initialization but a set capacity.
//
// NOTE: If no connections are provided via the `WithConnections` option,
// and the MaxCapacity is NOT set, the multiplexer will allocate a buffer
// of the default size which is 1.
func WithMaxCapacity(capacity int) Option {
	return func(m *Multiplexer) error {
		if capacity < 1 {
			return errInvalidMaxCapacity
		}

		m.capacity = capacity

		return nil
	}
}

// WithConnections provides an initial set of connections to be used by the
// multiplexer.
//
// NOTE: See note on `WithMaxCapacity` regarding the default buffer size when
// this option is used in conjunction with `WithMaxCapacity`, or no connections
// are provided via the `WithConnections` option.
func WithConnections(connections ...net.Conn) Option {
	return func(m *Multiplexer) error {
		var addr net.Addr
		for _, conn := range connections {
			if conn == nil {
				continue
			}

			if m.addr == nil {
				m.addr = conn.RemoteAddr()
			} else if m.addr.String() != conn.RemoteAddr().String() {
				return errAddrMismatch{
					Expected: addr,
					Actual:   conn.RemoteAddr(),
				}
			}

			m.initconns = append(m.initconns, conn)
		}

		return nil
	}
}

// WithConnector provides a function that can be used to create new connections
// in the multiplexer.
//
// NOTE: This method is only used when the current length of the multiplexer's
// connections is less than the multiplexer's capacity and a connection already
// in the multiplexer fails and must be recreated OR the multiplexer is
// configured for auto scaling.
func WithConnector(c Connect) Option {
	return func(m *Multiplexer) error {
		if c == nil {
			return errConnectorNil
		}

		m.connector = c

		return nil
	}
}

// WithAutoScaling sets a timeout on the multiplexer to automatically scale
// connections when a connection request exceeds the current the timeout when
// attempting to read from the internal buffer.
//
// NOTE: This options is ONLY available when a connector is provided via the
// `WithConnector` option.
//
// TODO: This needs to have the ability to set a timer for the new connection
// to be killed when the connection is no longer used.
func WithAutoScaling(timeout time.Duration) Option {
	return func(m *Multiplexer) error {
		if timeout <= 0 {
			return errInvalidTimeout
		}

		m.scaleTimeout = &timeout

		return nil
	}
}
