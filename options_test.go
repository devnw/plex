package plex

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"
)

func Test_WithReadWriters(t *testing.T) {
	testdata := map[string]struct {
		rws      []io.ReadWriter
		err      bool
		expected int
	}{
		"single": {
			[]io.ReadWriter{bytes.NewBuffer([]byte{})},
			false,
			1,
		},
		"single nil": {
			[]io.ReadWriter{},
			true,
			0,
		},
		"single valid, single nil (first)": {
			[]io.ReadWriter{
				nil,
				bytes.NewBuffer([]byte{}),
			},
			false,
			1,
		},
		"single valid, single nil (last)": {
			[]io.ReadWriter{
				bytes.NewBuffer([]byte{}),
				nil,
			},
			false,
			1,
		},
		"multi": {
			[]io.ReadWriter{
				bytes.NewBuffer([]byte{}),
				bytes.NewBuffer([]byte{}),
				bytes.NewBuffer([]byte{}),
				bytes.NewBuffer([]byte{}),
				bytes.NewBuffer([]byte{}),
			},
			false,
			5,
		},
		"multi nil": {
			[]io.ReadWriter{
				bytes.NewBuffer([]byte{}),
				nil,
				bytes.NewBuffer([]byte{}),
				nil,
				bytes.NewBuffer([]byte{}),
			},
			false,
			3,
		},
		"multi valid, single nil": {
			[]io.ReadWriter{
				bytes.NewBuffer([]byte{}),
				bytes.NewBuffer([]byte{}),
				nil,
				bytes.NewBuffer([]byte{}),
				bytes.NewBuffer([]byte{}),
			},
			false,
			4,
		},
	}

	for name, test := range testdata {
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			m, err := New(ctx, WithReadWriters(test.rws...))
			if err != nil {
				if !test.err {
					t.Fatal(err)
				}
				return
			}

			defer func() {
				err := m.Close()
				if err != nil {
					t.Errorf("Publisher.Close() failed: %v", err)
				}
			}()

			plex, ok := m.(*multiplexer)
			if !ok {
				t.Fatal("expected multiplexer")
			}

			if len(plex.readers) != test.expected {
				t.Fatalf("expected %d readers, got %d", test.expected, len(plex.readers))
			}

			if len(plex.writers) != test.expected {
				t.Fatalf("expected %d writers, got %d", test.expected, len(plex.writers))
			}
		})
	}
}

func Test_WithWriters(t *testing.T) {
	testdata := map[string]struct {
		ws       []io.Writer
		err      bool
		expected int
	}{
		"single": {
			[]io.Writer{bytes.NewBuffer([]byte{})},
			false,
			1,
		},
		"single nil": {
			[]io.Writer{},
			true,
			0,
		},
		"single valid, single nil (first)": {
			[]io.Writer{
				nil,
				bytes.NewBuffer([]byte{}),
			},
			false,
			1,
		},
		"single valid, single nil (last)": {
			[]io.Writer{
				bytes.NewBuffer([]byte{}),
				nil,
			},
			false,
			1,
		},
		"multi": {
			[]io.Writer{
				bytes.NewBuffer([]byte{}),
				bytes.NewBuffer([]byte{}),
				bytes.NewBuffer([]byte{}),
				bytes.NewBuffer([]byte{}),
				bytes.NewBuffer([]byte{}),
			},
			false,
			5,
		},
		"multi nil": {
			[]io.Writer{
				bytes.NewBuffer([]byte{}),
				nil,
				bytes.NewBuffer([]byte{}),
				nil,
				bytes.NewBuffer([]byte{}),
			},
			false,
			3,
		},
		"multi valid, single nil": {
			[]io.Writer{
				bytes.NewBuffer([]byte{}),
				bytes.NewBuffer([]byte{}),
				nil,
				bytes.NewBuffer([]byte{}),
				bytes.NewBuffer([]byte{}),
			},
			false,
			4,
		},
	}

	for name, test := range testdata {
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			m, err := New(ctx, WithWriters(test.ws...))
			if err != nil {
				if !test.err {
					t.Fatal(err)
				}
				return
			}

			defer func() {
				err := m.Close()
				if err != nil {
					t.Errorf("Publisher.Close() failed: %v", err)
				}
			}()

			plex, ok := m.(*multiplexer)
			if !ok {
				t.Fatal("expected multiplexer")
			}

			if len(plex.writers) != test.expected {
				t.Fatalf("expected %d writers, got %d", test.expected, len(plex.writers))
			}
		})
	}
}

func Test_WithReaders(t *testing.T) {
	testdata := map[string]struct {
		rs       []io.Reader
		err      bool
		expected int
	}{
		"single": {
			[]io.Reader{bytes.NewBuffer([]byte{})},
			false,
			1,
		},
		"single nil": {
			[]io.Reader{},
			true,
			0,
		},
		"single valid, single nil (first)": {
			[]io.Reader{
				nil,
				bytes.NewBuffer([]byte{}),
			},
			false,
			1,
		},
		"single valid, single nil (last)": {
			[]io.Reader{
				bytes.NewBuffer([]byte{}),
				nil,
			},
			false,
			1,
		},
		"multi": {
			[]io.Reader{
				bytes.NewBuffer([]byte{}),
				bytes.NewBuffer([]byte{}),
				bytes.NewBuffer([]byte{}),
				bytes.NewBuffer([]byte{}),
				bytes.NewBuffer([]byte{}),
			},
			false,
			5,
		},
		"multi nil": {
			[]io.Reader{
				bytes.NewBuffer([]byte{}),
				nil,
				bytes.NewBuffer([]byte{}),
				nil,
				bytes.NewBuffer([]byte{}),
			},
			false,
			3,
		},
		"multi valid, single nil": {
			[]io.Reader{
				bytes.NewBuffer([]byte{}),
				bytes.NewBuffer([]byte{}),
				nil,
				bytes.NewBuffer([]byte{}),
				bytes.NewBuffer([]byte{}),
			},
			false,
			4,
		},
	}

	for name, test := range testdata {
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			m, err := New(ctx, WithReaders(test.rs...))
			if err != nil {
				if !test.err {
					t.Fatal(err)
				}
				return
			}

			defer func() {
				err := m.Close()
				if err != nil {
					t.Errorf("Publisher.Close() failed: %v", err)
				}
			}()

			plex, ok := m.(*multiplexer)
			if !ok {
				t.Fatal("expected multiplexer")
			}

			if len(plex.readers) != test.expected {
				t.Fatalf("expected %d readers, got %d", test.expected, len(plex.readers))
			}
		})
	}
}

func Test_WithWriteBuffer(t *testing.T) {
	testdata := map[string]struct {
		buffer   int
		expected int
	}{
		"valid": {
			10,
			10,
		},
		"< 0": {
			-1,
			0,
		},
	}

	for name, test := range testdata {
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			m, err := New(
				ctx,
				WithWriteBuffer(test.buffer),
				WithReadWriters(bytes.NewBuffer([]byte{})),
			)
			if err != nil {
				t.Fatal(err)
			}

			defer func() {
				err := m.Close()
				if err != nil {
					t.Errorf("Publisher.Close() failed: %v", err)
				}
			}()

			plex, ok := m.(*multiplexer)
			if !ok {
				t.Fatal("expected multiplexer")
			}

			if plex.writeBufferSize != test.expected {
				t.Fatalf(
					"expected %d write buffer, got %d",
					test.expected,
					plex.writeBufferSize,
				)
			}
		})
	}
}

func Test_WithReadBuffer(t *testing.T) {
	testdata := map[string]struct {
		buffer   int
		expected int
	}{
		"valid": {
			10,
			10,
		},
		"< 0": {
			-1,
			0,
		},
	}

	for name, test := range testdata {
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			m, err := New(
				ctx,
				WithReadBuffer(test.buffer),
				WithReadWriters(bytes.NewBuffer([]byte{})),
			)
			if err != nil {
				t.Fatal(err)
			}

			defer func() {
				err := m.Close()
				if err != nil {
					t.Errorf("Publisher.Close() failed: %v", err)
				}
			}()

			plex, ok := m.(*multiplexer)
			if !ok {
				t.Fatal("expected multiplexer")
			}

			if plex.readBufferSize != test.expected {
				t.Fatalf(
					"expected %d write buffer, got %d",
					test.expected,
					plex.readBufferSize,
				)
			}
		})
	}
}

func WithErrOption() Option {
	return func(m *multiplexer) error {
		return errors.New("err")
	}
}

func Test_OptionError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_, err := New(
		ctx,
		WithErrOption(),
		WithReadWriters(bytes.NewBuffer([]byte{})),
	)

	if err == nil {
		t.Fatal("expected error")
	}
}
