// plex is a library for managing net.Conn connection types with minimal effort
// this library utilizes the options pattern to allow for easy configuration and
// customization of the multiplexer. This library provides multiple types which
// are used as wrappers to the configured net.Conn types which allow for general
// streaming of data through channels of bytes as an alternative to the
// more go traditional io.Reader/io.Writer interfaces.
//
// One important caveat is that the Plex library makes no assumptions of which
// net.Conn object is returned from either Reader or Writer. This means that
// messages must be routed by the consumer based on the data in the message
// rather than assuming that a response to a received message will be sent to
// the same connection which initiated the message round trip transmission.
//
// The goal of this library is to abstract the complexity of managing multiple
// connections and their associated channels of data into a single API which
// consumers can request Reader and Writer types from. These types implement
// the io.Closer method which will re-queue the Reader or Writer to the correct
// internal buffer within the multiplexer for another consumer down the line.
package plex

import (
	"context"
	"net"
	"sync"
	"time"
)

// TODO: Add support for auto-scaling and self-healing of failed connections

// New creates a net.Conn Multiplexer with the provided options.
//
// CRITICAL: A Multiplexer must be closed by calling the Close method when it
// is no longer needed. Failure to do so may result in goroutines leaking.
//
// NOTE: A single instance of the Multiplexer can ONLY be used to manage a
// set of connections for a SINGLE host.
//
// NOTE: It is important to read the documentation for the options which you
// wish to use to ensure proper configuration of the multiplexer.
func New(ctx context.Context, opts ...Option) (*Multiplexer, error) {
	ctx, cancel := _ctx(ctx)

	m := &Multiplexer{
		ctx:    ctx,
		cancel: cancel,
	}

	// Apply mutliplexer options
	for _, opt := range opts {
		err := opt(m)
		if err != nil {
			return nil, err
		}
	}

	// Ensure that the capacity for this multiplexer is correct
	// for the number of streams it will be handling.
	// DEFAULT: 1
	capacity := m.capacity
	if capacity == 0 && len(m.initconns) == 0 {
		capacity = 1
	} else if capacity == 0 {
		capacity = len(m.initconns)
	} else if capacity < len(m.initconns) {
		return nil, errTooManyConns
	}

	// Initialize the internal channels
	m.readers = make(chan Conn, capacity)
	m.writers = make(chan Conn, capacity)

	// Add the valid connections to the multiplexer
	// and clean up the init
	_, err := m.Add(m.ctx, m.initconns...)
	if err != nil {
		return nil, err
	}

	// Ensure proper configuration for auto-scaling
	if m.scaleTimeout != nil && m.connector == nil {
		return nil, errImproperAutoScalingNilConnector
	}

	// nil out the init connections
	// so that they are not accidentally
	// added to the multiplexer again
	m.initconns = nil

	return m, nil
}

// Multiplexer is the type which manages the multiplexing of net.Conn
// types through the use of channels and goroutines.
type Multiplexer struct {
	ctx          context.Context
	cancel       context.CancelFunc
	addr         net.Addr
	readers      chan Conn
	writers      chan Conn
	connector    Connect
	capacity     int
	scaleTimeout *time.Duration

	initconns []Conn
}

// Addr returns the net.Addr of the connections in this instance of the
// multiplexer.
func (m *Multiplexer) Addr() net.Addr {
	return m.addr
}

// Close closes the multiplexer and all of the connections it manages.
func (m *Multiplexer) Close() (err error) {
	defer func() {
		err = recoverErr(err, recover())
	}()

	m.cancel()
	<-m.ctx.Done()

	// Kill the streams
	m.kill(m.readers)
	m.kill(m.writers)

	return err
}

// kill drains the internal multiplexer channels and closes the underlying
// connections that it contains
func (m *Multiplexer) kill(conns chan Conn) {
	if conns == nil {
		return
	}

	defer func() { _ = recover() }() // ignore any panics

	for {
		select {
		case <-m.ctx.Done():
			return
		case s, ok := <-conns:
			if !ok {
				return
			}

			if s == nil {
				continue
			}

			// TODO: handle error here eventually
			_ = s.Close()
		}
	}
}

// Add adds the provided net.Conn instances to the multiplexer.
//
// NOTE: If the Multiplexer does not have the capacity to add all of the
// provided connections, then the connections that were unable to be added
// will be returned. It is possible the the connection was able to be added
// for a reader or writer, or both but will not indicate which.
//
// TODO: Correct this in the future, we should be able to know if it was
// added to one of them or both.
func (m *Multiplexer) Add(
	ctx context.Context,
	conns ...Conn,
) (left []Conn, err error) {
	defer func() {
		err = recoverErr(err, recover())
	}()

	ctx = merge(m.ctx, ctx)

	var i int
	for i = 0; i < len(conns); i++ {
		c := conns[i]

		select {
		case <-m.ctx.Done():
			return nil, m.ctx.Err()
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			if c == nil {
				continue
			}

			if m.addr == nil {
				m.addr = c.RemoteAddr()
			} else if c.RemoteAddr().Network() != m.Addr().Network() ||
				c.RemoteAddr().String() != m.Addr().String() {
				return conns[i:], errAddrMismatch{
					Expected: m.addr,
					Actual:   c.RemoteAddr(),
				}
			}

			left = m.add(ctx, m.readers, c)
			left = append(left, m.add(ctx, m.writers, c)...)
		}
	}

	return dedup(left), err
}

// dedup removes any duplicate connections from the provided slice of
// connections and returns the result.
func dedup(in []Conn) (out []Conn) {
	m := make(map[Conn]struct{})
	for _, c := range in {
		m[c] = struct{}{}
	}

	for c := range m {
		out = append(out, c)
	}

	return out
}

// add adds the provided connection to the provided channel.
// this is separated out so that it can correctly handle channels
// passed to it rather than having extra duplicate code. This also
// adheres to both contexts and breaks down the code a bit.
func (m *Multiplexer) add(
	ctx context.Context,
	streams chan Conn,
	conns ...Conn,
) []Conn {
	ctx = merge(m.ctx, ctx)

	var i int
	// Add as many as possible to the stream
connloop:
	for i = 0; i < len(conns) && cap(streams) > len(streams); i++ {
		select {
		case <-m.ctx.Done():
			break connloop
		case <-ctx.Done():
			break connloop
		case streams <- conns[i]:
		}
	}

	return conns[i:]
}

// Reader returns a wrapped net.Conn object as a Reader which provides
// stream capabilities to the net.Conn as well as cuts down on the available
// methods on the net.Conn. This is done on purpose to ensure that the
// consumer is less likely to misuse the Reader when requesting one for the
// purpose of reading.
// nolint:dupl
func (m *Multiplexer) Reader(
	ctx context.Context,
	timeout *time.Duration,
) (Reader, error) {
	ctx = merge(m.ctx, ctx)
	var tchan <-chan time.Time

	if timeout != nil {
		timer := time.NewTimer(*timeout)
		defer timer.Stop()
		tchan = timer.C
	}

	select {
	case <-m.ctx.Done():
		return nil, m.ctx.Err()
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-tchan:
		return nil, errTimeout
	case conn, ok := <-m.readers:
		if !ok {
			return nil, errClosed
		}

		rctx, rcancel := _ctx(ctx)

		return &readStream{
			conn,             // Conn
			rctx,             // context
			rcancel,          // cancel
			sync.WaitGroup{}, // wg
			sync.Mutex{},     // mutex
			// cleanup
			func() { // TODO: this may need to take the conn for self-heal
				_ = m.add(m.ctx, m.readers, conn)
			},
		}, nil
	}
}

// Writer returns a wrapped net.Conn object as a Writer which provides
// stream capabilities to the net.Conn as well as cuts down on the available
// methods on the net.Conn. This is done on purpose to ensure that the
// consumer is less likely to misuse the Writer when requesting one for the
// purpose of writing.
// nolint:dupl
func (m *Multiplexer) Writer(
	ctx context.Context,
	timeout *time.Duration,
) (Writer, error) {
	ctx = merge(m.ctx, ctx)
	var tchan <-chan time.Time

	if timeout != nil {
		timer := time.NewTimer(*timeout)
		defer timer.Stop()
		tchan = timer.C
	}

	select {
	case <-m.ctx.Done():
		return nil, m.ctx.Err()
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-tchan:
		return nil, errTimeout
	case conn, ok := <-m.writers:
		if !ok {
			return nil, errClosed
		}

		wctx, wcancel := _ctx(ctx)

		return &writeStream{
			conn,             // Conn
			wctx,             // context
			wcancel,          // cancel
			sync.WaitGroup{}, // wg
			sync.Mutex{},     // mutex
			// cleanup
			func() { // TODO: this may need to take the conn for self-heal
				_ = m.add(m.ctx, m.writers, conn)
			},
		}, nil
	}
}
