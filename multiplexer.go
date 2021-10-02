package plex

import (
	"context"
	"io"
	"time"
)

// multiplexer is the internal implementation of the Multiplexer interface which
// is used to multiplex multiple io.Readers and io.Writers into a single channel
// each for reading and writing. This struct and associated methods handle the
// internal state of the multiplexer and are not intended to be used directly.
type multiplexer struct {
	ctx    context.Context
	cancel context.CancelFunc

	readBufferSize  int
	writeBufferSize int

	// These init slices are only used by the options and New method
	// when instantiating the multiplexer the first time. They are used
	// to calculate the size of the internal buffers as well as allow
	// for all the options to be configured for the multiplexer prior
	// to initialization.
	initReadPool      []interface{}
	initWritePool     []interface{}
	initReadWritePool []interface{}

	readers chan ReadStream
	writers chan WriteStream

	// Used to add new readers and writers to the multiplexer in the event
	// of a failure on a reader/writer.
	initializer Initializer
}

// unexported method force internal only implementations of Multiplexer
func (m *multiplexer) isPlex() {}

// Close closes the multiplexer and all of its streams.
func (m *multiplexer) Close() (err error) {
	defer func() {
		err = recoverErr(recover())
	}()

	m.cancel()
	<-m.ctx.Done()

	return err
}

// cleanup handles the cleanup of the multiplexer when the context is canceled.
func (m *multiplexer) cleanup() {
	go func() {
		// Clean up the internal channels
		defer close(m.readers)
		defer close(m.writers)

		<-m.ctx.Done()

		// Drain the readers channel and close them
		for stream := range m.readers {
			stream.Close()
		}

		// Drain the writers channel and close them
		for stream := range m.writers {
			stream.Close()
		}
	}()
}

// Add adds any passed io.Reader/io.Writer, io.ReadWriter to the multiplexer.
func (m *multiplexer) Add(
	ctx context.Context,
	objs ...interface{},
) error {
	return m.queue(ctx, objs...)
}

// queue correctly handles the addition of supported streams or readers/writers
// to the multiplexer.
func (m *multiplexer) queue(
	ctx context.Context,
	in ...interface{},
) (err error) {
	defer func() {
		err = recoverErr(recover())
	}()

	for _, incoming := range in {
		select {
		case <-m.ctx.Done():
			return m.ctx.Err()
		case <-ctx.Done():
			return ctx.Err()
		default:
			switch value := incoming.(type) {
			case nil:
				return ErrNil
			case []interface{}:
				return m.queue(ctx, value...)
			case ReadStream:
				err := m.queueReadStreams(ctx, value)
				if err != nil {
					return err
				}
			case WriteStream:
				err := m.queueWriteStreams(ctx, value)
				if err != nil {
					return err
				}
			case []ReadStream:
				err := m.queueReadStreams(ctx, value...)
				if err != nil {
					return err
				}
			case []WriteStream:
				err := m.queueWriteStreams(ctx, value...)
				if err != nil {
					return err
				}
			case io.ReadWriter:
				err := m.queueReaders(ctx, value)
				if err != nil {
					return err
				}
				err = m.queueWriters(ctx, value)
				if err != nil {
					return err
				}
			case []io.ReadWriter:
				for _, rw := range value {
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
			case io.Reader:
				err := m.queueReaders(ctx, value)
				if err != nil {
					return err
				}
			case []io.Reader:
				err := m.queueReaders(ctx, value...)
				if err != nil {
					return err
				}
			case io.Writer:
				err := m.queueWriters(ctx, value)
				if err != nil {
					return err
				}
			case []io.Writer:
				err := m.queueWriters(ctx, value...)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (m *multiplexer) queueReadStreams(
	ctx context.Context,
	streams ...ReadStream,
) error {
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

func (m *multiplexer) queueReaders(
	ctx context.Context,
	readers ...io.Reader,
) error {
	// TODO: update to NewWriteStreams after nested context is implemented
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
	// TODO: update to NewWriteStreams after nested context is implemented
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

// Reader returns one of the multiplexed readers for reading. Reading should
// occur in completion (for the use case) by the caller before closing the
// returned reader. A closed reader is returned to the multiplexer's pool of
// available readers for reuse. If there is an internal error in a reader
// within the multiplexer's pool of available readers, the multiplexer will
// eliminate the reader from the pool and attempt to retrieve a new one if
// possible.
func (m *multiplexer) Reader(
	ctx context.Context,
	timeout time.Duration,
) (io.ReadCloser, error) {
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case <-m.ctx.Done():
		return nil, m.ctx.Err()
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-timer.C:
		return nil, ErrTimeout
	case r, ok := <-m.readers:
		if !ok {
			return nil, ErrClosed
		}

		ctx, cancel := context.WithCancel(ctx)

		return &reader{
			ctx:    ctx,
			cancel: cancel,
			r:      r,
			buffer: m.readBufferSize,
			cleanup: func() {
				// This is already a read stream so
				// no need to update the buffer
				m.queue(ctx, -1, r)
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
func (m *multiplexer) Writer(
	ctx context.Context,
	timeout time.Duration,
) (io.WriteCloser, error) {
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case <-m.ctx.Done():
		return nil, m.ctx.Err()
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-timer.C:
		return nil, ErrTimeout
	case w, ok := <-m.writers:
		if !ok {
			return nil, ErrClosed
		}

		ctx, cancel := context.WithCancel(ctx)

		return &writer{
			ctx:    ctx,
			cancel: cancel,
			w:      w,
			buffer: m.writeBufferSize,
			cleanup: func() {
				// This is already a read stream so
				// no need to update the buffer
				m.queue(ctx, -1, w)
			},
		}, nil
	}
}
