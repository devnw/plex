package plex

import (
	"context"
)

// New creates a new Multiplexer using the provided io.ReadWriters
func New(ctx context.Context, opts ...Option) (Multiplexer, error) {
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
	err := m.queueReaders(ctx, m.initReadPool...)
	if err != nil {
		return nil, err
	}

	err = m.queueWriters(ctx, m.initWritePool...)
	if err != nil {
		return nil, err
	}

	err = m.queueReadWriters(ctx, m.initReadWritePool...)
	if err != nil {
		return nil, err
	}

	// initialize cleanup routine
	go m.cleanup()

	return m, nil
}
