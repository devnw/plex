package plex

import (
	"bytes"
	"context"
	"crypto/sha1"
	"errors"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

// This is a stupid test, only for coverage
func Test_isPlex(t *testing.T) {
	(&multiplexer{}).isPlex()
}

func Test_Multiplexer_Reader(t *testing.T) {
	data, err := setOfRandBytes(100)
	if err != nil {
		t.Fatal(err)
	}

	for sum, test := range data {
		t.Logf("data length: %v bytes", len(test))

		t.Run(sum, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			m, err := New(
				ctx,
				WithReaders(bytes.NewBuffer(test)),
			)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			defer func() {
				err = m.Close()
				if err != nil {
					t.Errorf("Publisher.Close() failed: %v", err)
				}
			}()

			rc, err := m.Reader(ctx, nil)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			var output []byte
			var n int
			buff := make([]byte, 3)

			for err == nil {
				// Read from rc
				n, err = rc.Read(buff)

				// Append the read bytes to the output
				output = append(output, buff[:n]...)
			}

			if err != nil && err != io.EOF {
				t.Fatalf("unexpected error: %v", err)
			}

			rc.Close()

			if len(output) != len(test) {
				t.Fatalf("unexpected read length: %d", len(output))
			}

			diff := cmp.Diff(output, test)
			if diff != "" {
				t.Fatalf(
					"byte mismatch\n %s", diff,
				)
			}
		})
	}
}

func Test_Multiplexer_Multi_Reader(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dataset := map[string]bool{}
	data, err := setOfRandBytes(100)
	if err != nil {
		t.Fatal(err)
	}

	m, err := New(
		ctx,
		WithReadWriters(data.SliceOfReadWriter()...),
	)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	defer func() {
		err := m.Close()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}()

	for i := 0; i < len(data); i++ {
		rc, err := m.Reader(ctx, nil)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		var output []byte
		var n int
		buff := make([]byte, 3)

		for err == nil {
			// Read from rc
			n, err = rc.Read(buff)

			// Append the read bytes to the output
			output = append(output, buff[:n]...)
		}

		if err != nil && err != io.EOF {
			t.Fatalf("unexpected error: %v", err)
		}

		rc.Close()

		sum := fmt.Sprintf("%x", sha1.Sum(output))
		_, exists := data[sum]
		if !exists {
			t.Fatalf("unexpected sum: %v", sum)
		}

		seen := dataset[sum]
		if seen {
			t.Fatalf("duplicate sum: %v", sum)
		}

		// Mark the sum as seen
		dataset[sum] = true
		t.Logf("found sum: %v", sum)
	}

	for k := range data {
		seen, exists := dataset[k]
		if !exists || !seen {
			t.Fatalf("missing sum: %v", k)
		}
	}
}

func Test_Multplexer_Writer(t *testing.T) {
	data, err := setOfRandBytes(100)
	if err != nil {
		t.Fatal(err)
	}

	for sum, test := range data {
		t.Run(sum, func(t *testing.T) {
			t.Logf("data length: %v bytes", len(test))

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			stream := NewStream(ctx, 0)

			m, err := New(
				ctx,
				WithWriters(stream),
			)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			testdata := make([]byte, len(test))
			copy(testdata, test)

			go func(testdata []byte) {
				wc, err := m.Writer(ctx, nil)
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				defer wc.Close()

				var total int
				var read int
				for total < len(testdata) {
					read, err = wc.Write(testdata[read:])
					if err != nil {
						t.Errorf("unexpected error: %v", err)
						return
					}

					total += read
				}
			}(testdata)

			data := stream.Out(ctx)

		readloop:
			for i := 0; ; i++ {
				if i == len(test) {
					m.Close()
				}

				select {
				case <-ctx.Done():
					t.Fatalf("unexpected error: %v", ctx.Err())
				case bte, ok := <-data:
					if !ok {
						break readloop
					}

					if test[i] != bte {
						t.Fatalf(
							"byte mismatch at index %v; expected %v; got %v",
							i,
							test[i],
							bte,
						)
					}
				}
			}
		})
	}
}

func Test_Plex_AllReaders_Busy_RequestReader(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	m, err := New(
		ctx,
		WithReaders(
			bytes.NewBuffer([]byte("test1")),
			bytes.NewBuffer([]byte("test2")),
			bytes.NewBuffer([]byte("test3")),
		),
	)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Test 1
	test1, err := m.Reader(ctx, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Test 2
	test2, err := m.Reader(ctx, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Test 3
	test3, err := m.Reader(ctx, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	timeout := time.Millisecond * 100

	// Test 4 (should timeout)
	_, err = m.Reader(ctx, &timeout)
	if err != ErrTimeout {
		t.Errorf("unexpected error: %v", err)
	}

	// Free up the reader
	test1.Close()

	// Test 5
	test5, err := m.Reader(ctx, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	buff := make([]byte, 5)

	n, err := test5.Read(buff)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if n != 5 || string(buff[:n]) != "test1" {
		t.Fatalf("unexpected data: %v", string(buff[:n]))
	}

	// Free up the reader
	test2.Close()

	// Test 6
	test6, err := m.Reader(ctx, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	n, err = test6.Read(buff)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if n != 5 || string(buff[:n]) != "test2" {
		t.Fatalf("unexpected data: %v", string(buff[:n]))
	}

	// Free up the reader
	test3.Close()

	// Test 7
	test7, err := m.Reader(ctx, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	n, err = test7.Read(buff)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if n != 5 || string(buff[:n]) != "test3" {
		t.Fatalf("unexpected data: %v", string(buff[:n]))
	}
}

func poolReadMethod(
	ctx context.Context,
	t *testing.T,
	start <-chan struct{},
	m Multiplexer,
	sumchan chan<- string,
) {
	<-start

	reader, err := m.Reader(ctx, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	msg := make([]byte, 0)

	n := 0
	buff := make([]byte, 5)

	for err == nil {
		n, err = reader.Read(buff)

		msg = append(msg, buff[:n]...)

		if err != nil {
			break
		}
	}

	if err != io.EOF {
		t.Errorf("unexpected error: %v", err)
	}

	select {
	case <-ctx.Done():
		return
	case sumchan <- fmt.Sprintf("%x", sha1.Sum(msg)):
	}
}

// nolint:funlen
func Test_Plex_AllReaders_Busy_RequestReader_Parallel(t *testing.T) {
	testdata := []struct {
		poolsize   int
		readMethod func(
			ctx context.Context,
			t *testing.T,
			start <-chan struct{},
			m Multiplexer,
			sumchan chan<- string,
		)
	}{
		{
			10,
			poolReadMethod,
		},
		{
			20,
			poolReadMethod,
		},
		{
			50,
			poolReadMethod,
		},
		{
			100,
			poolReadMethod,
		},
		{
			500,
			poolReadMethod,
		},
		{
			1000,
			poolReadMethod,
		},
	}

	for _, test := range testdata {
		t.Run(fmt.Sprintf("parallel_poolsize_%v", test.poolsize), func(t *testing.T) {
			sums := make(map[string]bool)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			sets, err := setOfRandBytes(test.poolsize)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			// Setup the data harness
			readers := make([]io.Reader, len(sets))
			for sum, data := range sets {
				sums[sum] = false

				readers = append(readers, bytes.NewBuffer(data))
			}

			m, err := New(
				ctx,
				WithReaders(readers...),
			)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			start := make(chan struct{})
			sumchan := make(chan string, len(sums))

			for i := 0; i < test.poolsize; i++ {
				go poolReadMethod(ctx, t, start, m, sumchan)
			}

			close(start)

			for i := 0; i < len(sums); i++ {
				select {
				case <-ctx.Done():
					t.Fatalf("unexpected error: %v", ctx.Err())
					return
				case sum, ok := <-sumchan:
					if !ok {
						t.Fatal("unexpected error: channel closed")
						return
					}

					if val, ok := sums[sum]; !ok || val {
						if val {
							t.Fatalf("unexpected duplicate sum: %v", sum)
						}

						t.Fatalf("unexpected sum: %v", sum)
						return
					}

					sums[sum] = true
				}
			}

			for k, v := range sums {
				if !v {
					t.Fatalf("expected missing sum: %v", k)
				}
			}
		})
	}
}

// func Test_Out(t *testing.T) {
// 	testdata := map[string]struct {
// 		rwc      *rwc
// 		expected []byte
// 		err      bool
// 	}{
// 		"valid": {
// 			&rwc{
// 				buffer: make([]byte, 5),
// 				wrote:  make(chan bool),
// 			},
// 			[]byte("test1"),
// 			false,
// 		},
// 	}

// 	for name, test := range testdata {
// 		t.Run(name, func(t *testing.T) {
// 			ctx, cancel := context.WithCancel(context.Background())
// 			defer cancel()

// 			plex := New(ctx, 0, WithReadWriters(test.rwc))
// 			defer plex.Close()
// 		})
// 	}
// }

// func Test_In(t *testing.T) {
// 	testdata := map[string]struct {
// 		rwc *rwc
// 		err bool
// 	}{
// 		"valid": {
// 			&rwc{
// 				buffer: []byte("test1"),
// 			},
// 			false,
// 		},
// 	}

// 	for name, test := range testdata {
// 		t.Run(name, func(t *testing.T) {
// 			ctx, cancel := context.WithCancel(context.Background())
// 			defer cancel()

// 			multi := New(ctx, 0, WithReadWriters(test.rwc))

// 			select {
// 			case <-ctx.Done():
// 				return
// 			case data, ok := <-multi.In():
// 				if !ok {
// 					t.Fatalf("expected success")
// 				}

// 				if !reflect.DeepEqual(test.rwc.buffer, data) {
// 					t.Fatalf("Expected [%s]; got [%s]", string(test.rwc.buffer), string(test.rwc.buffer))
// 				}
// 			}
// 		})
// 	}
// }

// func Test_In_Parallel(t *testing.T) {
// 	testdata := []int{
// 		1,
// 		10,
// 		100,
// 		1000,
// 		10000,
// 	}

// 	input := &rwc{
// 		buffer: []byte("test1"),
// 	}

// 	for _, test := range testdata {
// 		t.Run(fmt.Sprintf("%v", test), func(t *testing.T) {
// 			ctx, cancel := context.WithCancel(context.Background())
// 			defer cancel()

// 			multi, err := New(
// 				ctx,
// 				input,
// 				// input,
// 				// input,
// 				// input,
// 				// input,
// 			)
// 			if err != nil {
// 				t.Fatalf("expected success; %s", err)
// 			}

// 			data := make(chan []byte, test)
// 			hold := make(chan bool)

// 			var wg sync.WaitGroup
// 			wg.Add(test)

// 			for i := 0; i < test; i++ {
// 				t.Logf("LOOP EXEC %v", i+1)
// 				go func() {
// 					defer wg.Done()
// 					<-hold

// 					// buff := make([]byte, 10)
// 					// read, err := multi.Read(buff)
// 					// if err != nil {
// 					// 	t.Error(err)
// 					// }

// 					// t.Logf("READ %v bytes", read)

// 					// data <- buff
// 					select {
// 					case <-ctx.Done():
// 						return
// 					case data <- <-multi.In():
// 						fmt.Println("pushed")
// 					}
// 				}()
// 			}

// 			go func() {
// 				wg.Wait()
// 				fmt.Println("closing data")
// 				close(data)
// 				multi.Close()
// 			}()

// 			close(hold)
// 			var count int
// 		dloop:
// 			for {
// 				select {
// 				case <-ctx.Done():
// 					t.Fatal(ctx.Err())
// 				case _, ok := <-data:
// 					if !ok {
// 						break dloop
// 					}
// 					count++
// 				}
// 			}

// 			if count != test {
// 				t.Fatalf("Expected %v; got %v", test, count)
// 			}
// 		})
// 	}
// }

func Test_multiplexer_Add_canceled(t *testing.T) {
	for name, test := range ctxCancelTests() {
		t.Run(name, func(t *testing.T) {
			m := &multiplexer{ctx: test.parent}
			test.Eval(t, m.Add(test.child, &wStream{}))
		})
	}
}

func Test_multiplexer_Reader_canceled(t *testing.T) {
	for name, test := range ctxCancelTests() {
		t.Run(name, func(t *testing.T) {
			m := &multiplexer{ctx: test.parent}

			timeout := time.Second
			_, err := m.Reader(test.child, &timeout)

			test.Eval(t, err)
		})
	}
}

func Test_multiplexer_Writer_canceled(t *testing.T) {
	for name, test := range ctxCancelTests() {
		t.Run(name, func(t *testing.T) {
			m := &multiplexer{ctx: test.parent}

			timeout := time.Second
			_, err := m.Writer(test.child, &timeout)

			test.Eval(t, err)
		})
	}
}

func evalErr(err error) error {
	if err == nil {
		return errors.New("Expected error")
	}

	if err != context.Canceled {
		return fmt.Errorf("Expected context.Canceled; got %v", err)
	}

	return nil
}

// NOTE: These still contain a possible race because the select could choose
// the non-ctx Done option, but running it 100000 times like this didn't fail
// the tests
func Test_multiplexer_New_Writer_canceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var err error
	for i := 0; i < 100; i++ {
		_, err = New(
			ctx,
			WithWriters(bytes.NewBuffer([]byte("test"))),
		)

		err = evalErr(err)
		if err == nil {
			return
		}
	}

	if err != nil {
		t.Error(err)
	}
}

func Test_multiplexer_New_Reader_canceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var err error
	for i := 0; i < 100; i++ {
		_, err = New(
			ctx,
			WithReaders(bytes.NewBuffer([]byte("test"))),
		)

		err = evalErr(err)
		if err == nil {
			return
		}
	}

	if err != nil {
		t.Error(err)
	}
}

func Test_multiplexer_Reader_timeout(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	m := &multiplexer{ctx: ctx}

	timeout := time.Millisecond
	_, err := m.Reader(ctx, &timeout)
	if err != ErrTimeout {
		t.Errorf("Expected ErrTimeout; got %v", err)
	}
}

func Test_multiplexer_Reader_closed_readers(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	readers := make(chan ReadStream)

	m := &multiplexer{
		ctx:     ctx,
		readers: readers,
	}

	close(readers)

	_, err := m.Reader(ctx, nil)
	if err != ErrClosed {
		t.Errorf("Expected ErrClosed; got %v", err)
	}
}

func Test_multiplexer_Reader_cleanup(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	m, err := New(
		ctx,
		WithReaders(bytes.NewBuffer([]byte("test"))),
	)
	if err != nil {
		t.Fatal(err)
	}

	r, err := m.Reader(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}

	multiplex, ok := m.(*multiplexer)
	if !ok {
		t.Fatal("Expected multiplexer")
	}

	if len(multiplex.readers) != 0 {
		t.Errorf("Expected 0 readers; got %v", len(multiplex.readers))
	}

	r.Close()

	if len(multiplex.readers) != 1 {
		t.Errorf("Expected 1 readers; got %v", len(multiplex.readers))
	}
}

func Test_multiplexer_Reader_cleanup_ctxcan(t *testing.T) {
	for i := 0; i < 100; i++ {
		ctx, cancel := context.WithCancel(context.Background())

		m, err := New(
			ctx,
			WithReaders(bytes.NewBuffer([]byte("test"))),
		)
		if err != nil {
			t.Fatal(err)
		}

		r, err := m.Reader(ctx, nil)
		if err != nil {
			t.Fatal(err)
		}

		cancel()
		err = r.Close()
		if err == context.Canceled {
			return
		}
	}

	t.Fatal("Expected context.Canceled")
}

func Test_multiplexer_Writer_timeout(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	m := &multiplexer{ctx: ctx}

	timeout := time.Millisecond
	_, err := m.Writer(ctx, &timeout)
	if err != ErrTimeout {
		t.Errorf("Expected ErrTimeout; got %v", err)
	}
}

func Test_multiplexer_Writer_closed_writers(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	m := &multiplexer{
		ctx:     ctx,
		writers: make(chan WriteStream),
	}

	close(m.writers)

	_, err := m.Writer(ctx, nil)
	if err != ErrClosed {
		t.Errorf("Expected ErrClosed; got %v", err)
	}
}

func Test_multiplexer_Writer_cleanup(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	m, err := New(
		ctx,
		WithWriters(bytes.NewBuffer([]byte("test"))),
	)
	if err != nil {
		t.Fatal(err)
	}

	w, err := m.Writer(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}

	multiplex, ok := m.(*multiplexer)
	if !ok {
		t.Fatal("Expected multiplexer")
	}

	if len(multiplex.writers) != 0 {
		t.Errorf("Expected 0 writers; got %v", len(multiplex.writers))
	}

	w.Close()

	if len(multiplex.writers) != 1 {
		t.Errorf("Expected 1 writers; got %v", len(multiplex.writers))
	}
}

func Test_multiplexer_Writer_cleanup_ctxcan(t *testing.T) {
	for i := 0; i < 100; i++ {
		ctx, cancel := context.WithCancel(context.Background())

		m, err := New(
			ctx,
			WithWriters(bytes.NewBuffer([]byte("test"))),
		)
		if err != nil {
			t.Fatal(err)
		}

		w, err := m.Writer(ctx, nil)
		if err != nil {
			t.Fatal(err)
		}

		cancel()
		err = w.Close()
		if err == context.Canceled {
			return
		}
	}

	t.Fatal("Expected context.Canceled")
}

func Test_multiplexer_New_ReaderWriter_canceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var err error
	for i := 0; i < 100; i++ {
		_, err = New(
			ctx,
			WithReadWriters(bytes.NewBuffer([]byte("test"))),
		)

		err = evalErr(err)
		if err == nil {
			return
		}
	}

	if err != nil {
		t.Error(err)
	}
}
