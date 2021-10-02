package plex

import "io"

// Option is the function literate that is used to configure the multiplexer
// during initialization.
type Option func(*multiplexer) error

// TODO: define what buffer does for these options

// WithReadBuffer adds a buffer to the multiplexer's io.Readers.
// NOTE: This also applies to the io.Reader implementation of an io.ReadWriter.
func WithReadBuffer(size int) Option {
	return func(m *multiplexer) error {
		m.readBufferSize = size
		return nil
	}
}

// WithWriteBuffer adds a buffer to the multiplexer's io.Writers.
// NOTE: This also applies to the io.Writer implementation of an io.ReadWriter.
func WithWriteBuffer(size int) Option {
	return func(m *multiplexer) error {
		m.writeBufferSize = size
		return nil
	}
}

// WithReaders is a convenience function to add multiple Readers to the
// multiplexer.
func WithReaders(rs ...io.Reader) Option {
	return func(m *multiplexer) error {
		for _, r := range rs {
			if r == nil {
				continue
			}

			m.initReadPool = append(m.initReadPool, r)
		}

		return nil
	}
}

// WithWriters is a convenience function to add multiple Writers to the
// multiplexer.
func WithWriters(ws ...io.Writer) Option {
	return func(m *multiplexer) error {
		for _, w := range ws {
			if w == nil {
				continue
			}

			m.initWritePool = append(m.initWritePool, w)
		}

		return nil
	}
}

// WithReadWriters is a convenience function to add multiple ReadWriters to the
// multiplexer.
func WithReadWriters(rws ...io.ReadWriter) Option {
	return func(m *multiplexer) error {
		for _, rw := range rws {
			if rw == nil {
				continue
			}

			m.initReadWritePool = append(m.initReadWritePool, rw)
		}

		return nil
	}
}

// WithInitializer gives the multiplexer a chance to re-initialize it's internal
// state by executing the given function and passing it the multiplexer queue
// method where it will be able to add new readers and writers.
func WithInitializer(init Initializer) Option {

	// TODO:
	return func(m *multiplexer) error {
		m.initializer = init
		return nil
	}
}
