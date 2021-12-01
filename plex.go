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

	defer close(m.readers)
	defer close(m.writers)

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

// queue correctly handles the addition of supported streams or readers/writers
// to the multiplexer.
// nolint:gocyclo
func (m *multiplexer) queue(
	ctx context.Context,
	in ...interface{},
) (err error) {
	defer func() {
		err = recoverErr(err, recover())
	}()
	ctx = merge(m.ctx, ctx)

	for _, incoming := range in {
		select {
		case <-m.ctx.Done():
			return m.ctx.Err()
		case <-ctx.Done():
			return ctx.Err()
		default:
			var err error

			switch value := incoming.(type) {
			case nil:
				return ErrNil
			case []interface{}:
				return m.queue(ctx, value...)
			case ReadStream:
				err = m.queueReadStreams(ctx, value)
			case WriteStream:
				err = m.queueWriteStreams(ctx, value)
			case []ReadStream:
				err = m.queueReadStreams(ctx, value...)
			case []WriteStream:
				err = m.queueWriteStreams(ctx, value...)
			case io.ReadWriter:
				err = m.queueReadWriters(ctx, value)
			case []io.ReadWriter:
				err = m.queueReadWriters(ctx, value...)
			case io.Reader:
				err = m.queueReaders(ctx, value)
			case []io.Reader:
				err = m.queueReaders(ctx, value...)
			case io.Writer:
				err = m.queueWriters(ctx, value)
			case []io.Writer:
				err = m.queueWriters(ctx, value...)
			}

			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (m *multiplexer) queueReadStreams(
	ctx context.Context,
	streams ...ReadStream,
) error {
	ctx = merge(m.ctx, ctx)

	for _, stream := range streams {
		select {
		case <-m.ctx.Done():
			return m.ctx.Err()
		case <-ctx.Done():
			return ctx.Err()
		case m.readers <- stream:
		}
	}

	return nil
}

func (m *multiplexer) queueWriteStreams(
	ctx context.Context,
	streams ...WriteStream,
) error {
	ctx = merge(m.ctx, ctx)

	for _, stream := range streams {
		select {
		case <-m.ctx.Done():
			return m.ctx.Err()
		case <-ctx.Done():
			return ctx.Err()
		case m.writers <- stream:
		}
	}

	return nil
}

// TODO: Should there be an attempt to track
// the count of available readers and writers?

func (m *multiplexer) queueReaders(
	ctx context.Context,
	readers ...io.Reader,
) error {
	ctx = merge(m.ctx, ctx)

	for _, r := range readers {
		select {
		case <-m.ctx.Done():
			return m.ctx.Err()
		case <-ctx.Done():
			return ctx.Err()
		case m.readers <- NewReadStream(
			m.ctx,
			r,
			m.readBufferSize,
		):
		}
	}

	return nil
}

func (m *multiplexer) queueWriters(
	ctx context.Context,
	writers ...io.Writer,
) error {
	ctx = merge(m.ctx, ctx)

	for _, w := range writers {
		select {
		case <-m.ctx.Done():
			return m.ctx.Err()
		case <-ctx.Done():
			return ctx.Err()
		case m.writers <- NewWriteStream(
			m.ctx,
			w,
			m.writeBufferSize,
		):
		}
	}

	return nil
}

func (m *multiplexer) queueReadWriters(
	ctx context.Context,
	rws ...io.ReadWriter,
) error {
	ctx = merge(m.ctx, ctx)

	for _, rw := range rws {
		select {
		case <-m.ctx.Done():
			return m.ctx.Err()
		case <-ctx.Done():
			return ctx.Err()
		default:
			err := m.queueReaders(ctx, rw)
			if err != nil {
				return err
			}

			err = m.queueWriters(ctx, rw)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// Reader returns one of the multiplexed readers for reading. Reading should
// occur in completion (for the use case) by the caller before closing the
// returned reader. A closed reader is returned to the multiplexer's pool of
// available readers for reuse. If there is an internal error in a reader
// within the multiplexer's pool of available readers, the multiplexer will
// eliminate the reader from the pool and attempt to retrieve a new one if
// possible.
// nolint:dupl
func (m *multiplexer) Reader(
	ctx context.Context,
	timeout *time.Duration,
) (io.ReadCloser, error) {
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

		rctx, rcancel := context.WithCancel(ctx)

		return &reader{
			ctx:    rctx,
			cancel: rcancel,
			r:      r,
			buffer: m.readBufferSize,
			cleanup: func() error {
				select {
				case <-m.ctx.Done():
				case <-rctx.Done():
				default:
					// This is already a read stream so
					// no need to update the buffer
					return m.queue(ctx, -1, r)
				}

				return nil
			},
		}, nil
	}
}

// Writer returns one of the multiplexed writers for writing. Writing should
// occur in completion by the caller (for the use case) before closing the
// returned writer. A closed writer is returned to the multiplexer's pool of
// available writers for reuse. If there is an internal error in a writer
// within the multiplexer's pool of available writers, the multiplexer will
// eliminate the writer from the pool and attempt to retrieve a new one if
// possible.
// nolint:dupl
func (m *multiplexer) Writer(
	ctx context.Context,
	timeout *time.Duration,
) (io.WriteCloser, error) {
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

		wctx, wcancel := context.WithCancel(ctx)

		return &writer{
			ctx:    wctx,
			cancel: wcancel,
			w:      w,
			buffer: m.writeBufferSize,
			cleanup: func() error {
				select {
				case <-m.ctx.Done():
				case <-wctx.Done():
				default:
					// This is already a read stream so
					// no need to update the buffer
					return m.queue(ctx, -1, w)
				}

				return nil
			},
		}, nil
	}
}
