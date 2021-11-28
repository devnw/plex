package plex

import (
	"bytes"
	"context"
	"crypto/sha1"
	"fmt"
	"testing"
	"time"
)

func Test_Streams_Read(t *testing.T) {
	data, err := setOfRandBytes(100)
	if err != nil {
		t.Fatal(err)
	}

	for _, test := range data {
		t.Logf("data length: %v bytes", len(test))

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		out := Read(ctx, bytes.NewBuffer(test), len(test))

		var i int

	readloop:
		for ; ; i++ {
			select {
			case <-ctx.Done():
				t.Fatal("context done")
			case b, ok := <-out:
				if !ok {
					break readloop
				}

				if b != test[i] {
					t.Fatalf("expected %v, got %v", test[i], b)
				}
			}
		}

		if i != len(test) {
			t.Fatalf("expected %v, got %v", len(test), i)
		}
	}
}

func Benchmark_Streams_Read(b *testing.B) {
	data, err := randBytes(1)
	if err != nil {
		b.Fatal(err)
	}

	bte := data[0]

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	rws := NewStream(ctx, 0)
	defer func() {
		err := rws.Close()
		if err != nil {
			b.Errorf("stream close error: %v", err)
		}
	}()

	// Read the data being written to the stream
	go func() {
		// Data transfer channels
		write := rws.In(ctx)
		for {
			select {
			case <-ctx.Done():
				return
			case write <- bte:
			}
		}
	}()
	read := rws.Out(ctx)

	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		select {
		case <-ctx.Done():
			return
		case <-read:
		}
	}
}

func Test_Streams_Write(t *testing.T) {
	data, err := setOfRandBytes(100)
	if err != nil {
		t.Fatal(err)
	}

	for _, test := range data {
		t.Logf("data length: %v bytes", len(test))

		t.Run(fmt.Sprintf("%x", sha1.Sum(test)), func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			rws := NewStream(ctx, 0)
			defer func() {
				err := rws.Close()
				if err != nil {
					t.Errorf("stream close error: %v", err)
				}
			}()

			// Data transfer channels
			read := rws.Out(ctx)

			go func(rws Stream) {
				// Ignoring this is fine because if it panic's due to an
				// issue with a closed channel that is correct, if it doesn't
				// the test will fail because there won't be enough bytes
				// to satisfy the read.
				defer func() {
					_ = recover()
				}()

				write := rws.In(ctx)
				defer close(write)

				for _, b := range test {
					select {
					case <-ctx.Done():
						return
					case write <- b:
					}
				}
			}(rws)

			var i int
		readloop:
			for i = 0; i < len(test); i++ {
				select {
				case <-ctx.Done():
					t.Fatalf("context done | %s", ctx.Err())
				case b, ok := <-read:
					if !ok {
						break readloop
					}

					if b != test[i] {
						t.Fatalf("expected byte %v, got %v", test[i], b)
					}
				}
			}
		})
	}
}

func Benchmark_Streams_Write(b *testing.B) {
	data, err := randBytes(1)
	if err != nil {
		b.Fatal(err)
	}

	bte := data[0]

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	rws := NewStream(ctx, 0)
	defer func() {
		err := rws.Close()
		if err != nil {
			b.Errorf("stream close error: %v", err)
		}
	}()

	// Read the data being written to the stream
	go func() {
		// Data transfer channels
		read := rws.Out(ctx)
		for {
			select {
			case <-ctx.Done():
				return
			case <-read:
			}
		}
	}()

	write := rws.In(ctx)

	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		select {
		case <-ctx.Done():
			return
		case write <- bte:
		}
	}
}

func Test_Write(t *testing.T) {
	data, err := setOfRandBytes(100)
	if err != nil {
		t.Fatal(err)
	}

	for sum, test := range data {
		t.Logf("data length: %v bytes", len(test))

		t.Run(sum, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// Create a concurrent safe writer
			stream := NewStream(ctx, 0)

			go func() {
				// Ignoring this is fine because if it panic's due to an
				// issue with a closed channel that is correct, if it doesn't
				// the test will fail because there won't be enough bytes
				// to satisfy the read.
				defer func() {
					_ = recover()
				}()

				wchan := Write(ctx, stream, 0)
				defer close(wchan)

				for _, b := range test {
					select {
					case <-ctx.Done():
						t.Error("context done")
					case wchan <- b:
					}
				}
			}()

			out := stream.Out(ctx)
		readloop:
			for i := 0; i < len(test); i++ {
				select {
				case <-ctx.Done():
					t.Fatal("context done")
				case bte, ok := <-out:
					if !ok {
						break readloop
					}

					if bte != test[i] {
						t.Fatalf("expected byte %v, got %v", test[i], bte)
					}
				}
			}
		})
	}
}

type badWriter struct{}

func (b *badWriter) Write(p []byte) (n int, err error) {
	return 0, fmt.Errorf("bad writer error")
}

func Test_Write_Writer_Error(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wchan := Write(ctx, &badWriter{}, 0)

	timer := time.NewTimer(time.Millisecond * 50)

	for i := 0; i < 2; i++ {
		select {
		case <-ctx.Done():
			return
		case wchan <- 0:
		case <-timer.C:
			return
		}
	}

	t.Fatal("expected timer to return test")
}

func Test_Read(t *testing.T) {
	data, err := setOfRandBytes(100)
	if err != nil {
		t.Fatal(err)
	}

	for sum, test := range data {
		t.Logf("data length: %v bytes", len(test))

		t.Run(sum, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// Create a concurrent safe writer
			stream := NewStream(ctx, 0)

			go func() {
				defer func() {
					err := stream.Close()
					if err != nil {
						t.Error(err)
					}
				}()
				in := stream.In(ctx)

				for _, b := range test {
					select {
					case <-ctx.Done():
						t.Error("context done")
					case in <- b:
					}
				}
			}()

			out := Read(ctx, stream, 0)

		readloop:
			for i := 0; i < len(test); i++ {
				select {
				case <-ctx.Done():
					t.Fatal("context done")
				case bte, ok := <-out:
					if !ok {
						break readloop
					}

					if bte != test[i] {
						t.Fatalf("expected byte %v, got %v", test[i], bte)
					}
				}
			}
		})
	}
}
