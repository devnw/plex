package plex

import (
	"context"
	"crypto/sha1"
	"fmt"
	"sync"
	"testing"
)

func Test_stream_Recv(t *testing.T) {
	data, err := setOfRandBytes(100)
	if err != nil {
		t.Fatal(err)
	}

	for _, test := range data {
		t.Logf("data length: %v bytes", len(test))

		t.Run(fmt.Sprintf("%x", sha1.Sum(test)), func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			rws := newStream(ctx)
			defer func() {
				err = rws.Close()
				if err != nil {
					t.Errorf("stream close error: %v", err)
				}
			}()

			testdata := make([]byte, len(test))
			copy(testdata, test)

			go func(c *testconn, testdata []byte) {
				defer func() {
					_ = recover()
				}()

				for _, b := range testdata {
					select {
					case <-ctx.Done():
						return
					case c.rwc.data <- b:
					}
				}

				close(c.rwc.data)
			}(rws, testdata)

			conctx, cancel := context.WithCancel(ctx)
			s := &readStream{
				rws.rwc,
				conctx,           // context
				cancel,           // cancel
				sync.WaitGroup{}, // wg
				sync.Mutex{},     // mutex

				// cleanup
				func() {},
			}

			defer func() {
				err = s.Close()
				if err != nil {
					t.Errorf("stream close error: %v", err)
				}
			}()

			read, err := s.Recv(ctx, 0)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

		readloop:
			for i := 0; i < len(test); i++ {
				select {
				case <-ctx.Done():
					t.Fatalf("context done | %s", ctx.Err())
				case b, ok := <-read:
					if !ok {
						break readloop
					}

					if b != test[i] {
						t.Fatalf("expected byte %v, got %v @ index %v", test[i], b, i)
					}
				}
			}
		})
	}
}

func Test_stream_Send(t *testing.T) {
	data, err := setOfRandBytes(100)
	if err != nil {
		t.Fatal(err)
	}

	for _, test := range data {
		t.Logf("data length: %v bytes", len(test))

		t.Run(fmt.Sprintf("%x", sha1.Sum(test)), func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			rws := newStream(ctx)
			defer func() {
				err = rws.Close()
				if err != nil {
					t.Errorf("stream close error: %v", err)
				}
			}()

			conctx, cancel := context.WithCancel(ctx)
			s := &writeStream{
				rws.rwc,
				conctx,           // context
				cancel,           // cancel
				sync.WaitGroup{}, // wg
				sync.Mutex{},     // mutex
				// cleanup
				func() {},
			}

			defer func() {
				err = s.Close()
				if err != nil {
					t.Errorf("stream close error: %v", err)
				}
			}()

			testdata := make([]byte, len(test))
			copy(testdata, test)

			go func(w Writer, testdata []byte) {
				defer func() {
					_ = recover()
				}()

				data, err := w.Send(ctx, 0)
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}

				for _, b := range testdata {
					select {
					case <-ctx.Done():
						return
					case data <- b:
					}
				}
				close(data)
			}(s, testdata)

			read, err := rws.rwc.Recv(ctx, 0)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

		readloop:
			for i := 0; i < len(test); i++ {
				select {
				case <-ctx.Done():
					t.Fatalf("context done | %s", ctx.Err())
				case b, ok := <-read:
					if !ok {
						break readloop
					}

					if b != test[i] {
						t.Fatalf("expected byte %v, got %v @ index %v", test[i], b, i)
					}
				}
			}
		})
	}
}
