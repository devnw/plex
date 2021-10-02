package plex

import (
	"bytes"
	"context"
	"io"
	"reflect"
	"testing"
	"time"
)

func Test_Reader(t *testing.T) {
	testdata := map[string]struct {
		rwc      *rwc
		expected []byte
		err      bool
	}{
		"valid single": {
			&rwc{
				buffer: make([]byte, 5),
				wrote:  make(chan bool),
			},
			[]byte("test1"),
			false,
		},
	}

	for name, test := range testdata {
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			m, err := New(
				ctx,
				1,
				WithReaders(bytes.NewBuffer(test.expected)),
			)
			if err != nil {
				if !test.err {
					t.Errorf("unexpected error: %v", err)
				}

				return
			}

			wc, err := m.Writer(ctx, time.Second)
			if err != nil {
				if !test.err {
					t.Errorf("unexpected error: %v", err)
				}

				return
			}

			rc, err := m.Reader(ctx, time.Second)
			if err != nil {
				if !test.err {
					t.Errorf("unexpected error: %v", err)
				}

				return
			}

			wc.Write(test.expected)

			lenexp := len(test.expected)
			t.Logf("lenexp: %v", lenexp)
			buff := make([]byte, lenexp)
			n, err := rc.Read(buff)
			if err != nil && err != io.EOF {
				if !test.err {
					t.Fatalf("unexpected error: %v", err)
				}

				return
			}

			rc.Close()

			if n != len(test.expected) {
				t.Fatalf("unexpected read length: %d", n)
			}

			if !reflect.DeepEqual(buff, test.expected) {
				t.Fatalf("unexpected read data: %v", buff)
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
