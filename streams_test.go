package plex

import (
	"bytes"
	"context"
	"crypto/sha1"
	"fmt"
	"testing"
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
	data, err := randBytes(100)
	if err != nil {
		b.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		out := Read(ctx, bytes.NewBuffer(data), len(data))

		var i int

	readloop:
		for ; ; i++ {
			select {
			case <-ctx.Done():
				b.Fatal("context done")
			case bte, ok := <-out:
				if !ok {
					break readloop
				}

				if bte != data[i] {
					b.Fatalf("expected %v, got %v", data[i], b)
				}
			}
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

			rws := NewReadWriteStream(ctx, 0)

			// Data transfer channels
			read := rws.Out(ctx)

			go func(rws ReadWriteStream) {
				defer func() {
					err := rws.Close()
					if err != nil {
						t.Error(err)
					}
				}()

				write := rws.In(ctx)
				defer close(write)

				for _, b := range test {
					select {
					case <-ctx.Done():
						t.Errorf("context done | %s", ctx.Err())
					case write <- b:
					}
				}
			}(rws)

		readloop:
			for i := 0; ; i++ {
				select {
				case <-ctx.Done():
					t.Fatalf("context done | %s", ctx.Err())
				case b, ok := <-read:
					if !ok {
						break readloop
					}

					if i >= len(test) {
						t.Fatalf(
							"expected len %v, got %v; byte %v",
							len(test),
							i,
							b,
						)
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
	data, err := randBytes(100)
	if err != nil {
		b.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		rws := NewReadWriteStream(ctx, 0)

		// Data transfer channels
		read := rws.Out(ctx)

		go func(rws ReadWriteStream) {
			defer func() {
				err := rws.Close()
				if err != nil {
					b.Error(err)
				}
			}()

			write := rws.In(ctx)
			defer close(write)

			for _, bte := range data {
				select {
				case <-ctx.Done():
					b.Errorf("context done | %s", ctx.Err())
				case write <- bte:
				}
			}
		}(rws)

	readloop:
		for i := 0; ; i++ {
			select {
			case <-ctx.Done():
				b.Fatalf("context done | %s", ctx.Err())
			case bte, ok := <-read:
				if !ok {
					break readloop
				}

				if i >= len(data) {
					b.Fatalf(
						"expected len %v, got %v; byte %v",
						len(data),
						i,
						b,
					)
				}

				if bte != data[i] {
					b.Fatalf("expected byte %v, got %v", data[i], bte)
				}
			}
		}
	}
}
