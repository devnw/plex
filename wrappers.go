package plex

type writer struct {
	WriteStream
	cleanup func() error
}

func (w *writer) Close() (err error) {
	if w.cleanup != nil {
		return w.cleanup()
	}

	return err
}

type reader struct {
	ReadStream
	cleanup func() error
}

func (r *reader) Close() (err error) {
	if r.cleanup != nil {
		return r.cleanup()
	}

	return err
}
