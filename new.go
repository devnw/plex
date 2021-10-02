package plex

import (
	"context"
)

// New creates a new Multiplexer using the provided io.ReadWriters
func New(ctx context.Context, buffer int, opts ...Option) (Multiplexer, error) {
	ctx, cancel := context.WithCancel(ctx)

	m := &multiplexer{
		ctx:    ctx,
		cancel: cancel,
	}

	// Apply mutliplexer options
	for _, opt := range opts {
		err := opt(m)
		if err != nil {
			return nil, err
		}
	}

	// Initialize readers channel
	m.readers = make(
		chan ReadStream,
		len(m.initReadPool)+len(m.initReadWritePool),
	)

	m.writers = make(
		chan WriteStream,
		len(m.initWritePool)+len(m.initReadWritePool),
	)

	// Queue up the pool of readers, writers and readwriters
	m.queue(
		m.ctx,
		append(
			m.initReadPool,
			append(
				m.initWritePool,
				m.initReadWritePool...,
			)...)...)

	return m, nil
}
