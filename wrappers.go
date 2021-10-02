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
	cleanup func()
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

func (w *writer) Close() error {
	if w.cleanup != nil {
		defer w.cleanup()
	}

	w.cancel()

	return nil
}

type reader struct {
	ctx     context.Context
	cancel  context.CancelFunc
	cleanup func()
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

func (r *reader) Close() error {
	if r.cleanup != nil {
		defer r.cleanup()
	}

	r.cancel()

	return nil
}
