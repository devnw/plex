// plex is a multiplexing library for io.Reader, io.Writer, and io.ReadWriter
// types which allows for requesting a reader or writer without needing to know
// which one is actually being used. This is useful for things like managing
// communication over a full-duplex TCP connection using a net.Conn type which
// implements the io.ReadWriter interface.
//
// Along with multiplexing using the io.Reader and io.Writer interfaces, plex
// also supports streaming using the ReadStream, WriteStream, and
// ReadWriteStream interfaces. These interfaces provide access to a single
// stream of data, which can be read or written to one byte at a time.
//
// Most types in plex are implement io.Closer and it is expected that the user
// properly close the streams upon completion using a `defer` statement or
// appropriate closing mechanism to ensure that the streams are returned to
// the pool of available streams.
package plex

import (
	"context"
	"io"
	"sync"
	"time"
)

// multiplexer is the internal implementation of the Multiplexer interface which
// is used to multiplex multiple io.Readers and io.Writers into a single channel
// each for reading and writing. This struct and associated methods handle the
// internal state of the multiplexer and are not intended to be used directly.
type multiplexer struct {
	ctx    context.Context
	cancel context.CancelFunc
	plexWg sync.WaitGroup
	plexMu sync.Mutex

	readBufferSize  int
	writeBufferSize int

	// These init slices are only used by the options and New method
	// when instantiating the multiplexer the first time. They are used
	// to calculate the size of the internal buffers as well as allow
	// for all the options to be configured for the multiplexer prior
	// to initialization.
	initReadPool      []io.Reader
	initWritePool     []io.Writer
	initReadWritePool []io.ReadWriter

	readers chan ReadStream
	writers chan WriteStream
}

// unexported method force internal only implementations of Multiplexer
func (m *multiplexer) isPlex() {}

// Close closes the multiplexer and all of its streams.
func (m *multiplexer) Close() (err error) {
	defer func() {
		err = recoverErr(err, recover())
	}()

	m.cancel()
	<-m.ctx.Done()

	m.plexMu.Lock()
	defer m.plexMu.Unlock()

	m.plexWg.Wait()

	return err
}

// Add adds any passed io.Reader/io.Writer, io.ReadWriter to the multiplexer.
func (m *multiplexer) Add(
	ctx context.Context,
	objs ...interface{},
) error {
	ctx = merge(m.ctx, ctx)

	return m.queue(ctx, objs...)
}

// Reader returns one of the multiplexed streams for reading. Reading
// should occur in completion by the caller (for the use case) before closing
// the returned ReadStream. A closed ReadStream is returned to the
// multiplexer's pool of available ReadStreams for reuse. If there is an
// internal error in a ReadStream within the multiplexer's pool of available
// readers, the multiplexer will eliminate the reader from the pool.
// nolint:dupl
func (m *multiplexer) Reader(
	ctx context.Context,
	timeout *time.Duration,
) (ReadStream, error) {
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
		return nil, ErrTimeout
	case r, ok := <-m.readers:
		if !ok {
			return nil, ErrClosed
		}

		return &reader{
			r,
			func() error {
				select {
				case <-m.ctx.Done():
					return m.ctx.Err()
				default:
					// This is already a read stream so
					// no need to update the buffer
					return m.queue(ctx, r)
				}
			},
		}, nil
	}
}

// Writer returns one of the multiplexed streams for writing. Writing
// should occur in completion by the caller (for the use case) before closing
// the returned WriteStream. A closed WriteStream is returned to the
// multiplexer's pool of available WriteStreams for reuse. If there is an
// internal error in a WriteStream within the multiplexer's pool of available
// writers, the multiplexer will eliminate the writer from the pool.
// nolint:dupl
func (m *multiplexer) Writer(
	ctx context.Context,
	timeout *time.Duration,
) (WriteStream, error) {
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
		return nil, ErrTimeout
	case w, ok := <-m.writers:
		if !ok {
			return nil, ErrClosed
		}

		return &writer{
			w,
			func() error {
				select {
				case <-m.ctx.Done():
					return m.ctx.Err()
				default:
					// This is already a read stream so
					// no need to update the buffer
					return m.queue(ctx, w)
				}
			},
		}, nil
	}
}
