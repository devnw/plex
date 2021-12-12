package plex

import (
	"context"
	"io"
	"sync"
)

// Interface enforcer
var _ Writer = (*writeStream)(nil)
var _ Reader = (*readStream)(nil)

// readStream wraps an io.ReadCloser and provides a set of methods for
// reading from the the underlying reader. It also bypasses the close method
// so that the stream can be closed but the reader can be peristed by the
// creator of the stream through a closure. The Multiplexer uses this feature
// to ensure it properly re-queues the stream when this stream is closed so
// that is it available to be used again.
type readStream struct {
	io.ReadCloser
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	mu      sync.Mutex
	cleanup func()
}

// Read is a passthrough to the underlying io.Reader with the exception of
// a failure in the underlying io.Reader which causes it to no longer be viable
// in that case an appropriate error is returned.
//
// TODO: Define and export this error
func (s *readStream) Read(b []byte) (int, error) {
	// TODO: this needs to adhere to the context so that it can
	// be exited in the event the stream is closed due to an error
	// and a new connection is established
	n, err := s.ReadCloser.Read(b)

	return n, err
}

// Recv provides a read-only channel for reading bytes from the underlying
// io.Reader. This is useful for situations where streaming data is desired
// rather than buffering through the use of the io.Reader interface. This
// method accepts a context as well as a buffer size. The buffer size is used
// to set the internal buffers for the returned channel as well as the buffer
// used to read data off the underlying io.Reader.
func (s *readStream) Recv(ctx context.Context, buffer int) (<-chan byte, error) {
	ctx, _ = _ctx(ctx)
	out := make(chan byte, buffer)

	go func(out chan<- byte, r io.Reader, buffer int) {
		defer func() {
			_ = recover() // TODO: handle in the future?
		}()

		defer func() {
			_ = s.Kill() // TODO: should this kill the underlying stream?
		}()

		defer close(out)

		// Cannot have a buffer less than 1 when calling a reader
		if buffer < 1 {
			buffer = 1
		}

		// Create the buffer to read into
		// Buffer doesn't need to be re-created
		// on each read so this buffer is shared
		// and the number of bytes read to the buffer
		// is returned from the io.Reader based
		// on the API defined in the go doc for io.Reader
		b := make([]byte, buffer)

		for {
			select {
			case <-ctx.Done():
				return
			default:
				read, err := r.Read(b)
				if err != nil {
					// TODO: handle error
					return
				}

				// Take each buffer value and push it to the stream
				for _, v := range b[:read] {
					select {
					case <-ctx.Done():
						return
					case out <- v:
					}
				}
			}
		}
	}(out, s, buffer)

	return out, nil
}

func (s *readStream) Kill() (err error) {
	defer func() {
		err = recoverErr(err, recover())
	}()

	defer func() {
		err = recoverErr(err, s.ReadCloser.Close())
	}()

	s.mu.Lock()
	// Bypass cleanup since we are killing the stream
	s.cleanup = nil
	s.mu.Unlock()

	err = s.Close()

	return err
}

func (s *readStream) Close() (err error) {
	defer func() {
		err = recoverErr(err, recover())
	}()

	s.cancel()
	<-s.ctx.Done()

	s.mu.Lock()
	defer s.mu.Unlock()

	defer func() {
		if s.cleanup == nil {
			return
		}
		s.cleanup()
	}()

	s.wg.Wait()

	return nil
}

// writeStream wraps an io.WriteCloser and provides a set of methods for
// writing from the the underlying writer. It also bypasses the close method
// so that the stream can be closed but the writer can be peristed by the
// creator of the stream through a closure. The Multiplexer uses this feature
// to ensure it properly re-queues the stream when this stream is closed so
// that is it available to be used again.
type writeStream struct {
	io.WriteCloser
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	mu      sync.Mutex
	cleanup func()
}

func (s *writeStream) Write(b []byte) (int, error) {
	// TODO: this needs to adhere to the context so that it can
	// be exited in the event the stream is closed due to an error
	// and a new connection is established
	n, err := s.WriteCloser.Write(b)

	return n, err
}

// Send provides a write-only channel for writing bytes to the underlying
// io.Writer. This is useful for situations where streaming data is desired
// rather than buffering through the use of the io.Writer interface. This
// method accepts a context as well as a buffer size. The buffer size is used
// to set the internal buffers for the returned channel as well as the buffer
// used to write data to the underlying io.Writer.
func (s *writeStream) Send(ctx context.Context, buffer int) (chan<- byte, error) {
	ctx, _ = _ctx(ctx)
	out := make(chan byte)

	go func(out chan byte, w io.Writer) {
		defer func() {
			_ = recover() // TODO: handle in the future?
		}()

		defer func() {
			_ = s.Kill()
		}()

		if buffer < 1 {
			buffer = 1
		}

		buff := make([]byte, buffer)
		index := 0
		for {
			select {
			case <-ctx.Done():
				return
			case b, ok := <-out:
				if ok {
					buff[index] = b
					index++
				}

				if index == buffer || !ok {
					out := make([]byte, len(buff[:index]))
					copy(out, buff[:index])

					_, _ = w.Write(out)
					// TODO: handle error
					// TODO: handle n

					buff = make([]byte, buffer)
					index = 0
				}

				if !ok {
					return
				}
			}
		}
	}(out, s)

	return out, nil
}

func (s *writeStream) Kill() (err error) {
	defer func() {
		err = recoverErr(err, recover())
	}()

	defer func() {
		err = recoverErr(err, s.WriteCloser.Close())
	}()

	s.mu.Lock()
	// Bypass cleanup since we are killing the stream
	s.cleanup = nil
	s.mu.Unlock()

	err = s.Close()

	return err
}

func (s *writeStream) Close() (err error) {
	defer func() {
		err = recoverErr(err, recover())
	}()

	s.cancel()
	<-s.ctx.Done()

	s.mu.Lock()
	defer s.mu.Unlock()

	defer func() {
		if s.cleanup == nil {
			return
		}
		s.cleanup()
	}()

	s.wg.Wait()

	return nil
}
