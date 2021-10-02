package plex

import (
	"context"
	"io"
	"time"
)

type Initializer interface {
	New(ctx context.Context) (interface{}, error)
}

type Adder interface {
	Add(ctx context.Context, objs ...interface{}) error
}

type unexported interface {
	// unexported method force internal only implementations of Multiplexer
	isPlex()
}

type Reader interface {
	unexported

	Adder

	Reader(context.Context, time.Duration) (io.ReadCloser, error)
}

type Writer interface {
	unexported

	Adder

	Writer(context.Context, time.Duration) (io.WriteCloser, error)
}

type ReadWriter interface {
	unexported

	Adder
	Reader
	Writer
}

// Multiplexer is a provider of Read/Write Closer and Read/Write Stream
// interfaces. It is the responsibility of the caller to close the
// returned Read/Write Closers in order to return them to the pool of available
// readers/writers. There is not a mechanism for returning them to the pool of
// available readers/writers without closing them.
type Multiplexer interface {
	unexported

	Reader
	Writer
	ReadWriter
	io.Closer
}

type ReadStream interface {
	io.Reader
	io.Closer
	Data(context.Context) <-chan byte
}

type WriteStream interface {
	io.Writer
	io.Closer
	Data(context.Context) chan<- byte
}

type ReadWriteStream interface {
	io.Reader
	io.Writer
	io.Closer
	Out(context.Context) <-chan byte
	In(context.Context) chan<- byte
}
