package plex

import (
	"fmt"
	"io"
)

type rwc struct {
	wcursor int
	rcursor int
	buffer  []byte
	closed  bool
	wrote   chan bool
}

func (w *rwc) Read(b []byte) (i int, err error) {
	fmt.Println(len(b))
	fmt.Println("read called")
	max := len(b)
	if len(w.buffer) < len(b) {
		max = len(w.buffer)
	}

	for ; i < max && w.rcursor < len(w.buffer); i++ {
		b[i] = w.buffer[i]
		w.rcursor++
	}

	if w.rcursor >= len(w.buffer) {
		err = io.EOF
	}
	fmt.Printf("read returning %v - %s\n", i, b)

	return i + 1, err
}

func (w *rwc) Write(b []byte) (written int, err error) {
	defer func() {
		if w.wrote != nil {
			close(w.wrote)
		}
	}()

	if w.wcursor >= len(w.buffer) {
		return 0, io.ErrShortWrite
	}

	inlen := len(b)
	for ; w.wcursor < len(w.buffer); w.wcursor++ {
		if written >= inlen {
			break
		}

		w.buffer[w.wcursor] = b[written]

		written++
	}

	if w.wcursor == len(w.buffer) && written < inlen {
		err = io.ErrShortWrite
	}

	return written, err
}

func (w *rwc) Close() error {
	w.closed = true
	return nil
}
