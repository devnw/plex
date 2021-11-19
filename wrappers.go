package plex

import (
	"context"
	"io"
)

// writer wraps an io.Writer so that the multiplexing implementation can control
// elements of the sub-writer lifecycle.
type writer struct {
	ctx     context.Context
	cancel  context.CancelFunc
	cleanup func() error
	w       io.Writer
	buffer  int
}

func (w *writer) Write(b []byte) (int, error) {
	select {
	// TODO: deal with the parent context
	case <-w.ctx.Done():
		return 0, w.ctx.Err()
	default:
		return w.w.Write(b)
	}
}

func (w *writer) Close() (err error) {
	if w.cleanup != nil {
		defer func() {
			err = w.cleanup()
		}()
	}

	w.cancel()

	return nil
}

type reader struct {
	ctx     context.Context
	cancel  context.CancelFunc
	cleanup func() error
	r       io.Reader
	buffer  int
}

func (r *reader) Read(b []byte) (int, error) {
	select {
	// TODO: deal with the parent context
	case <-r.ctx.Done():
		defer r.Close()
		return 0, r.ctx.Err()
	default:
		return r.r.Read(b)
	}
}

func (r *reader) Close() (err error) {
	if r.cleanup != nil {
		defer func() {
			err = r.cleanup()
		}()
	}

	r.cancel()

	return nil
}
