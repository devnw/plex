package plex

import (
	"bytes"
	"context"
	"crypto/sha1"
	"fmt"
	"io"
	"testing"
)

func Test_writer_Write(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cancelledCtx, cancelledCtxcan := context.WithCancel(context.Background())
	cancelledCtxcan()

	testdata := map[string]struct {
		ctx      context.Context
		w        *bytes.Buffer
		expected []byte
		err      bool
	}{
		"valid write": {
			ctx,
			bytes.NewBuffer(nil),
			[]byte("valid"),
			false,
		},
		"valid write, canceled context": {
			cancelledCtx,
			bytes.NewBuffer(nil),
			[]byte("valid"),
			true,
		},
	}

	for name, test := range testdata {
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(test.ctx)
			defer cancel()

			w := &writer{
				ctx,
				cancel,
				func() error { return nil },
				test.w,
				len(test.expected),
			}

			n, err := w.Write(test.expected)
			if err != nil {
				if !test.err {
					t.Error(err)
				}

				return
			}

			if n != len(test.expected) {
				t.Fatalf("expected %d bytes, got %d", len(test.expected), n)
			}

			if !bytes.Equal(test.expected, test.w.Bytes()) {
				t.Errorf("expected [%s], got [%s]", test.expected, test.w.Bytes())
			}
		})
	}
}

func Test_writer_Write_Randoms(t *testing.T) {
	data, err := setOfRandBytes(150)
	if err != nil {
		t.Fatal(err)
	}

	for _, test := range data {
		t.Logf("data length: %v bytes", len(test))

		t.Run(fmt.Sprintf("%x", sha1.Sum(test)), func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			testWriter := bytes.NewBuffer(nil)

			w := &writer{
				ctx,
				cancel,
				func() error { return nil },
				testWriter,
				0,
			}

			n, err := w.Write(test)
			if err != nil {
				t.Fatal(err)
			}

			if n != len(test) {
				t.Fatalf("expected %d bytes, got %d", len(test), n)
			}

			if !bytes.Equal(test, testWriter.Bytes()) {
				t.Errorf("expected [%s], got [%s]", test, testWriter.Bytes())
			}
		})
	}
}

func Test_reader_Read_Randoms(t *testing.T) {
	data, err := setOfRandBytes(150)
	if err != nil {
		t.Fatal(err)
	}

	for _, test := range data {
		t.Logf("data length: %v bytes", len(test))

		t.Run(fmt.Sprintf("%x", sha1.Sum(test)), func(t *testing.T) {
			var err error
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			testReader := bytes.NewBuffer(test)

			r := &reader{
				ctx,
				cancel,
				func() error { return nil },
				testReader,
				0,
			}

			index := 0
			buf := make([]byte, 10)

			for err == nil {
				n := 0
				n, err = r.Read(buf)

				for i := 0; i < n; i++ {
					if buf[i] != test[index] {
						t.Errorf(
							"expected [%v], got [%v]",
							test[index],
							buf[i],
						)
					}

					index++
				}
			}

			if err != nil && err != io.EOF {
				t.Fatal(err)
			}

			if index != len(test) {
				t.Fatalf("expected %d bytes, got %d", len(test), index)
			}
		})
	}
}

// func Test_reader_Read(t *testing.T) {
// 	testdata := map[string]struct {
// 		r        io.Reader
// 		buff     []byte
// 		expected []byte
// 		n        int
// 		err      bool
// 	}{
// 		"": {},
// 	}

// 	for name, test := range testdata {
// 		t.Run(name, func(t *testing.T) {

// 		})
// 	}
// }
