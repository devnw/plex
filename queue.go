package plex

import (
	"context"
	"io"
)

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
				// TODO: Should this really error?
				err = ErrNil
			case []interface{}:
				err = m.queue(ctx, value...)
			case Stream, []Stream:
				// TODO: Add Support for this
				continue
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
		if stream == nil {
			return ErrNil
		}

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
		if stream == nil {
			return ErrNil
		}

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
		if r == nil {
			return ErrNil
		}

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
		if w == nil {
			return ErrNil
		}

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
		if rw == nil {
			return ErrNil
		}

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
