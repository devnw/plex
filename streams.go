package plex

import (
	"context"
	"fmt"
	"io"
)

func NewReadStreams(ctx context.Context, buffer int, readers ...io.Reader) []ReadStream {
	var streams []ReadStream

	for _, r := range readers {
		streams = append(streams, NewReadStream(ctx, r, buffer))
	}

	return streams
}

func NewWriteStreams(ctx context.Context, buffer int, writers ...io.Writer) []WriteStream {
	var streams []WriteStream

	for _, w := range writers {
		streams = append(streams, NewWriteStream(ctx, w, buffer))
	}

	return streams
}

// NewReadStream creates a new ReadStream from an io.Reader.
func NewReadStream(ctx context.Context, r io.Reader, buffer int) ReadStream {
	ctx, cancel := context.WithCancel(ctx)

	return &rStream{
		ctx:    ctx,
		cancel: cancel,
		r:      Read(ctx, r, buffer),
	}
}

// NewWriteStream creates a new WStream from an io.Writer.
func NewWriteStream(ctx context.Context, w io.Writer, buffer int) WriteStream {
	ctx, cancel := context.WithCancel(ctx)

	return &wStream{
		ctx:    ctx,
		cancel: cancel,
		w:      Write(ctx, w, buffer),
	}
}

// Read reads from an io.Reader and pushes to a byte channel
// one byte at a time. Returns a read-only byte channel.
func Read(ctx context.Context, r io.Reader, buffer int) <-chan byte {
	out := make(chan byte, buffer)

	go func(out chan<- byte, r io.Reader, buffer int) {
		defer func() {
			recoverErr(recover())
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

	}(out, r, buffer)

	return out
}

// Write writes from a byte channel to an io.Writer one byte at a time using
// an intermediary bufio.Writer. Returns a write-only byte channel.
func Write(ctx context.Context, w io.Writer, buffer int) chan<- byte {
	out := make(chan byte, buffer)

	go func(out chan byte, w io.Writer, buffer int) {
		defer func() {
			recoverErr(recover())
		}()

		// It's possible for this to be closed upstream
		// in the event of a panic above the caller which
		// writes to the channel will need to have a recover
		// to handle a pre-mature close
		defer close(out)

		for {
			select {
			case <-ctx.Done():
				return
			case b, ok := <-out:
				if !ok {
					return
				}

				n, err := w.Write([]byte{b})
				if err != nil {
					// TODO: Handle error
					fmt.Println(err)
					return
				}

				if n != 1 {
					// TODO: Handle error
					fmt.Printf("n != 1: %d\n", n)
					return
				}
			}
		}
	}(out, w, buffer)

	return out
}

func NewReadWriteStream(ctx context.Context, buffer int) ReadWriteStream {
	ctx, cancel := context.WithCancel(ctx)
	data := make(chan byte, buffer)

	readCtx, readCancel := context.WithCancel(ctx)
	writeCtx, writeCancel := context.WithCancel(ctx)

	return &rwStream{
		ctx:    ctx,
		cancel: cancel,
		data:   data,
		r:      &rStream{ctx: readCtx, cancel: readCancel, r: data},
		w:      &wStream{ctx: writeCtx, cancel: writeCancel, w: data},
	}
}

// Interface enforcer
var _ ReadWriteStream = (*rwStream)(nil)

type rwStream struct {
	ctx    context.Context
	cancel context.CancelFunc
	data   chan byte
	r      *rStream
	w      *wStream
}

func (rws *rwStream) Out(ctx context.Context) <-chan byte {
	return rws.r.Data(ctx)
}

func (rws *rwStream) In(ctx context.Context) chan<- byte {
	return rws.w.Data(ctx)
}

func (rws *rwStream) Read(p []byte) (n int, err error) {
	return rws.r.Read(p)
}

func (rws *rwStream) Write(p []byte) (n int, err error) {
	return rws.w.Write(p)
}

func (rws *rwStream) Close() (err error) {
	defer func() {
		err = recoverErr(recover())
	}()

	defer rws.cancel()

	err = rws.w.Close()
	if err != nil {
		// TODO:
		fmt.Printf("Error closing wstream %s", err)
	}

	err = rws.r.Close()
	if err != nil {
		// TODO:
		fmt.Printf("Error closing rstream %s", err)
	}

	return err
}

// Interface enforcer
var _ ReadStream = (*rStream)(nil)

// rStream is a stream of bytes to be used only for reading. In the background
// it is a read-only channel of bytes. It implements a reader for pulling data
// from the channel and populating the supplied []byte. If the channel is closed
// the Read method will return an io.EOF error.
type rStream struct {
	ctx    context.Context
	cancel context.CancelFunc
	r      <-chan byte
}

func (r *rStream) Close() (err error) {
	defer func() {
		err = recoverErr(recover())
		if err != nil {
			fmt.Println(err)
		}
	}()

	r.cancel()
	return nil
}

// Data returns a read-only channel of bytes which read from an underlying
// io.Reader. The reader is responsible for cancelling the context when finished
// reading from the stream which allows the stream to be added back to the pool
// of available read streams.
func (r *rStream) Data(ctx context.Context) <-chan byte {
	out := make(chan byte)

	go func(out chan<- byte) {
		defer func() {
		}()
		defer close(out)

		for {
			select {
			case <-r.ctx.Done():
				return
			case <-ctx.Done():
				return
			case b, ok := <-r.r: // Forward data
				if !ok {
					return
				}

				select {
				case <-r.ctx.Done():
					return
				case <-ctx.Done():
					return
				case out <- b:
				}
			}
		}
	}(out)

	return out
}

// Read is the RStream io.Reader implementation.
func (r *rStream) Read(p []byte) (n int, err error) {
	defer func() {
		err = recoverErr(recover())
	}()

	var ok bool
	var i int

	for i = range p {
		select {
		case <-r.ctx.Done():
			return i, r.ctx.Err()
		case p[i], ok = <-r.r:
			if !ok {
				// Only return i here because i+1 is not written to p in this case
				return i, io.EOF
			}
		}
	}

	// return i+1 for a proper 0 index byte count
	return i + 1, nil
}

// Interface enforcer
var _ WriteStream = (*wStream)(nil)

// wStream is a stream of bytes to be used only for writing. In the background
// it is a write-only channel of bytes. It implements a writer for pushing data
// from to the underlying channel using the supplied []byte. If the channel is
// closed externally the Write method will return the panic value as an error.
// If the consumer of the wStream fails to pull from the channel the Write method
// will block in the case the channel is unbuffered or the buffer of the channel
// is full.
type wStream struct {
	ctx    context.Context
	cancel context.CancelFunc
	w      chan<- byte
}

func (w *wStream) Close() (err error) {
	defer func() {
		err = recoverErr(recover())
	}()

	w.cancel()
	return nil
}

// Data returns a write-only channel of bytes which write to an underlying
// io.Writer. The consumer is responsible for closing the channel when finished
// writing to the stream. The context can also be used to cancel the stream for
// the consumer which called Data causing it to be added back to the pool
// of available writers.
func (w *wStream) Data(ctx context.Context) chan<- byte {
	in := make(chan byte) // TODO: add buffer support to API

	go func(in <-chan byte) {
		defer func() {
			_ = recover()
		}()

		for {
			select {
			case <-w.ctx.Done():
				defer close(w.w) // TODO: Move these to a defer func called on init
				return
			case <-ctx.Done():
				return
			case b, ok := <-in:
				if !ok {
					return
				}

				select {
				case <-w.ctx.Done():
					defer close(w.w)
					return
				case <-ctx.Done():
					return
				case w.w <- b:
				}
			}
		}
	}(in)

	return in
}

// Write is the WStream io.Writer implementation.
func (w *wStream) Write(p []byte) (n int, err error) {
	defer func() {
		err = recoverErr(recover())
	}()

	for _, b := range p {
		select {
		case <-w.ctx.Done():
			defer close(w.w)
			return n, w.ctx.Err()
		case w.w <- b:
			n++
		}
	}

	return n, nil
}
