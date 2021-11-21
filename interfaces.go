package plex

import (
	"context"
	"io"
	"time"
)

// Adder defines an interface for adding an io.Reader, io.Writer, or
// io.ReadWriter after the initial instantiation of the multiplexer.
type Adder interface {
	Add(ctx context.Context, objs ...interface{}) error
}

// unexported creates a compiler enforced method of tying internal interface
// implementations to the plex library so that users are unable to implement
// specific types causing misuse of the library.
type unexported interface {
	// unexported method force internal only implementations of Multiplexer
	isPlex()
}

// Reader defines an interface for requesting an io.ReadCloser from the
// multiplexer. This will return an available io.ReadCloser from the pool of
// io.ReadClosers within the multiplexer.
type Reader interface {
	unexported

	Adder

	// TODO: Should this be updated to return a <-chan io.ReadCloser instead?
	// this would allow for external libraries to handle scaling the multiplexer
	// using select contention instead of requiring the plex library to handle it.
	Reader(context.Context, *time.Duration) (io.ReadCloser, error)
}

// Writer defines an interface for requesting an io.WriteCloser from the
// multiplexer. This will return an available io.WriteCloser from the pool of
// io.WriteClosers within the multiplexer.
type Writer interface {
	unexported

	Adder

	// TODO: Should this be updated to return a <-chan io.WriteCloser instead?
	// this would allow for external libraries to handle scaling the multiplexer
	// using select contention instead of requiring the plex library to handle it.
	Writer(context.Context, *time.Duration) (io.WriteCloser, error)
}

// ReadWriter defines an interface for requesting either an io.ReadCloser, or
// an io.WriteCloser from the multiplexer. This will return an the first
// available of the expected type from the internal pool.
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

// ReadStream defines an interface which exposes a method for retrieving a
// direct read-only channel of bytes for reading from the stream. The
// expected behavior is that the channel returned from Data only reads from a
// single stream internally. Once the ReadStream is closed the channel should
// be closed as well.
//
// NOTE: Closing the channel is the responsibility of the ReadStream
// implementation. This differs from the API expected by the WriteStream
// interface.
type ReadStream interface {
	io.Reader
	io.Closer
	Data(context.Context) <-chan byte
}

// WriteStream defines an interface which exposes a method for retrieving a
// direct write-only channel of bytes for writing to the stream. The
// expected behavior is that the channel returned from Data only writes to a
// single stream internally. Once the WriteStream is closed the channel should
// be closed as well.
//
// NOTE: Closing the channel is the responsibility of the caller. This differs
// from the API expected by the ReadStream interface.
type WriteStream interface {
	io.Writer
	io.Closer
	Data(context.Context) chan<- byte
}

// Stream defines an interface which exposes a method for retrieving a
// direct read/write channel of bytes for reading and writing to a stream. The
// methods used to read and write to the stream are accessed via the In and Out
// methods. The Out method returns a readable channel of bytes and the In method
// returns a writeable channel of bytes.
type Stream interface {
	io.Reader
	io.Writer
	io.Closer
	Out(context.Context) <-chan byte
	In(context.Context) chan<- byte
}
