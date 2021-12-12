package plex

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/gob"
	"fmt"
	"io"
	"net"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

// TODO: Test different reader/writer where the reader is one buffer
// and the writer is a different buffer.
// TODO: Do for encoders, as well as Recv/Send methods

func Test_Multiplexer_Reader_canceled(t *testing.T) {
	for name, test := range ctxCancelTests() {
		t.Run(name, func(t *testing.T) {
			m := &Multiplexer{ctx: test.parent}

			timeout := time.Second
			_, err := m.Reader(test.child, &timeout)

			test.Eval(t, err)
		})
	}
}

func Test_Multiplexer_Add_canceled(t *testing.T) {
	for name, test := range ctxCancelTests() {
		t.Run(name, func(t *testing.T) {
			m := &Multiplexer{
				ctx:     test.parent,
				readers: make(chan Conn, 1),
				writers: make(chan Conn, 1),
			}

			_, err := m.Add(test.child, &iotestconn{
				bytes.NewBuffer([]byte{}),
				true,
				"0.0.0.0",
			})

			test.Eval(t, err)
		})
	}
}

func netConnToConn(nc ...net.Conn) []Conn {
	conns := make([]Conn, len(nc))
	for i, c := range nc {
		conns[i] = c
	}
	return conns
}

func Test_Multiplexer_Add(t *testing.T) {
	for name, test := range connectionTests() {
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			m, err := New(
				ctx,
				WithMaxCapacity(100),
			)
			if err != nil {
				t.Fatal(err)
			}

			defer func() {
				err = m.Close()
				if err != nil {
					t.Errorf("Close() failed: %v", err)
				}
			}()

			_, err = m.Add(ctx, netConnToConn(test.conns...)...)
			if err != nil {
				if !test.err {
					t.Fatal(err)
				}
				return
			}

			if len(m.readers) != test.expected {
				t.Fatalf("expected %d readers, got %d", test.expected, len(m.readers))
			}

			if len(m.writers) != test.expected {
				t.Fatalf("expected %d writers, got %d", test.expected, len(m.writers))
			}
		})
	}
}

func Test_Multiplexer_Add_readers_left(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	m, err := New(
		ctx,
		WithConnections(
			&testconn{},
		),
	)
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		err = m.Close()
		if err != nil {
			t.Errorf("Close() failed: %v", err)
		}
	}()

	conns := []Conn{
		&testconn{},
	}

	<-m.writers

	before := len(m.writers)

	left, err := m.Add(ctx, conns...)
	if err != nil {
		t.Fatal(err)
	}

	if len(m.writers) != before+1 {
		t.Fatalf("expected %d writers; got %d", before+1, len(m.writers))
	}

	if len(conns) != len(left) {
		t.Fatalf("expected %d left; got %d", len(conns), len(left))
	}

	for i, conn := range conns {
		if conn != left[i] {
			t.Fatalf("expected %v; got %v", conn, left[i])
		}
	}
}

func Test_Multiplexer_Add_writers_left(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	m, err := New(
		ctx,
		WithConnections(
			&testconn{},
		),
	)
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		err = m.Close()
		if err != nil {
			t.Errorf("Close() failed: %v", err)
		}
	}()

	conns := []Conn{
		&testconn{},
	}

	<-m.readers

	before := len(m.readers)

	left, err := m.Add(ctx, conns...)
	if err != nil {
		t.Fatal(err)
	}

	if len(m.readers) != before+1 {
		t.Fatalf("expected %d readers; got %d", before+1, len(m.readers))
	}

	if len(conns) != len(left) {
		t.Fatalf("expected %d left; got %d", len(conns), len(left))
	}

	for i, conn := range conns {
		if conn != left[i] {
			t.Fatalf("expected %v; got %v", conn, left[i])
		}
	}
}

func Test_multiplexer_add_canceled(t *testing.T) {
	for name, test := range ctxCancelTests() {
		t.Run(name, func(t *testing.T) {
			m := &Multiplexer{
				ctx: test.parent,
			}

			for i := 0; i < 100; i++ {
				conns := make(chan Conn, 1)

				left := m.add(test.child, conns, &iotestconn{
					bytes.NewBuffer([]byte{}),
					true,
					"0.0.0.0",
				})

				if len(left) != 0 {
					return
				}
			}

			t.Error("expected context canceled error")
		})
	}
}

func Test_add_toomanystreams(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	testdata := map[int]struct {
		conns []Conn
		left  int
		err   error
	}{
		0: {
			conns: make([]Conn, 1),
			left:  1,
			err:   nil,
		},
		1: {
			conns: make([]Conn, 5),
			left:  4,
			err:   nil,
		},
		100: {
			conns: make([]Conn, 100),
			left:  0,
			err:   nil,
		},
	}

	m := &Multiplexer{
		ctx: ctx,
	}

	for buffer, test := range testdata {
		t.Run(fmt.Sprintf("buffer-%v-conns-%v", buffer, len(test.conns)), func(t *testing.T) {
			conns := make(chan Conn, buffer)

			left := m.add(ctx, conns, test.conns...)

			if test.left != len(left) {
				t.Errorf("Expected %v left; got %v", test.left, len(left))
			}
		})
	}
}

func Test_Multiplexer_Writer_canceled(t *testing.T) {
	for name, test := range ctxCancelTests() {
		t.Run(name, func(t *testing.T) {
			m := &Multiplexer{ctx: test.parent}

			timeout := time.Second
			_, err := m.Writer(test.child, &timeout)

			test.Eval(t, err)
		})
	}
}

func Test_Multiplexer_Reader_timeout(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	m := &Multiplexer{ctx: ctx}

	timeout := time.Millisecond
	_, err := m.Reader(ctx, &timeout)
	if err != errTimeout {
		t.Errorf("Expected ErrTimeout; got %v", err)
	}
}

func Test_Multiplexer_Writer_timeout(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	m := &Multiplexer{ctx: ctx}

	timeout := time.Millisecond
	_, err := m.Writer(ctx, &timeout)
	if err != errTimeout {
		t.Errorf("Expected ErrTimeout; got %v", err)
	}
}

func Test_Multiplexer_Busy_RequestReader(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	m, err := New(
		ctx,
		WithConnections(
			&iotestconn{rw: bytes.NewBuffer([]byte("test1"))},
			&iotestconn{rw: bytes.NewBuffer([]byte("test2"))},
			&iotestconn{rw: bytes.NewBuffer([]byte("test3"))},
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
	if err != errTimeout {
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

func Test_Multiplexer_Busy_RequestWriter(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	buffer := bytes.NewBuffer([]byte("testn"))

	m, err := New(
		ctx,
		WithConnections(
			&iotestconn{rw: buffer},
			&iotestconn{rw: buffer},
			&iotestconn{rw: buffer},
		),
	)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Test 1
	test1, err := m.Writer(ctx, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Test 2
	test2, err := m.Writer(ctx, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Test 3
	test3, err := m.Writer(ctx, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	timeout := time.Millisecond * 100

	// Test 4 (should timeout)
	_, err = m.Writer(ctx, &timeout)
	if err != errTimeout {
		t.Errorf("unexpected error: %v", err)
	}

	// Free up the reader
	test1.Close()

	// Test 5
	test5, err := m.Writer(ctx, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	n, err := test5.Write([]byte("test1"))
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if n != 5 || !strings.HasSuffix(buffer.String(), "test1") {
		t.Fatalf("unexpected data: %v", buffer.String())
	}

	// Free up the reader
	test2.Close()

	// Test 6
	test6, err := m.Writer(ctx, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	n, err = test6.Write([]byte("test2"))
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if n != 5 || !strings.HasSuffix(buffer.String(), "test2") {
		t.Fatalf("unexpected data: %v", buffer.String())
	}

	// Free up the reader
	test3.Close()

	// Test 7
	test7, err := m.Writer(ctx, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	n, err = test7.Write([]byte("test3"))
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if n != 5 || !strings.HasSuffix(buffer.String(), "test3") {
		t.Fatalf("unexpected data: %v", string(buffer.Bytes()[:n]))
	}
}

func poolReadMethod(
	ctx context.Context,
	t *testing.T,
	start <-chan struct{},
	m *Multiplexer,
	sumchan chan<- string,
) {
	<-start

	reader, err := m.Reader(ctx, nil)
	if err != nil {
		return
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
		if err != context.Canceled {
			return
		}

		t.Errorf("unexpected error: %v", err)
	}

	select {
	case <-ctx.Done():
		return
	case sumchan <- fmt.Sprintf("%x", sha1.Sum(msg)):
	}
}

// nolint:funlen
func Test_Multiplexer_AllReaders_Busy_RequestReader_Parallel(t *testing.T) {
	testdata := []struct {
		poolsize   int
		readMethod func(
			ctx context.Context,
			t *testing.T,
			start <-chan struct{},
			m *Multiplexer,
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
			conns := make([]net.Conn, 0, len(sets))
			for sum, data := range sets {
				sums[sum] = false

				conns = append(conns, &iotestconn{rw: bytes.NewBuffer(data)})
			}

			m, err := New(
				ctx,
				WithConnections(conns...),
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

type MyType struct {
	I int
	B bool
	F float32
	S string
	M MyType2
}

type MyType2 struct {
	B bool
	I int32
}

func Test_Encoders(t *testing.T) {
	gob.Register(MyType{})
	gob.Register(MyType2{})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	m, err := New(
		ctx,
		WithConnections(&iotestconn{rw: bytes.NewBuffer([]byte{})}),
	)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	writer, err := m.Writer(ctx, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	reader, err := m.Reader(ctx, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	enc := gob.NewEncoder(writer)
	dec := gob.NewDecoder(reader)

	data := MyType{
		I: 1,
		B: true,
		F: 1.0,
		S: "test",
		M: MyType2{
			B: true,
			I: 1,
		},
	}

	count := 1000
	for i := 0; i < count; i++ {
		t.Logf("encoding %v", data)

		err := enc.Encode(data)
		if err != nil && err != io.EOF {
			t.Fatal(err)
		}

		v := MyType{}

		t.Logf("decoding type %T", v)
		err = dec.Decode(&v)
		if err != nil && err != io.EOF {
			t.Fatal(err)
		}

		if !reflect.DeepEqual(data, v) {
			t.Fatalf("unexpected data: %v", v)
		}

		t.Logf("data match")
	}
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
				WithConnections(&iotestconn{rw: bytes.NewBuffer(test)}),
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

type killstr struct {
	io.ReadWriter
	close func() error
}

func (*killstr) RemoteAddr() net.Addr {
	return &net.TCPAddr{
		IP:   net.IP{0, 0, 0, 0},
		Port: 0,
	}
}

func (k *killstr) Close() error {
	return k.close()
}

func Test_kill(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	conns := make(chan Conn, 2)
	done := make(chan struct{})

	conns <- nil
	conns <- &killstr{
		bytes.NewBuffer([]byte{}),
		func() error {
			close(done)
			return nil
		},
	}

	m, err := New(ctx)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	go m.kill(conns)

	timer := time.NewTimer(time.Second)
	select {
	case <-done:
	case <-timer.C:
		t.Fatal("timeout")
	}
}

func Test_kill_closed_chan(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	conns := make(chan Conn)

	m, err := New(ctx)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	close(conns)
	m.kill(conns)
}

func Test_kill_ctxcan(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	conns := make(chan Conn)

	m, err := New(ctx)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	cancel()
	m.kill(conns)
}

func Test_kill_panic(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	conns := make(chan Conn, 1)

	conns <- &killstr{
		bytes.NewBuffer([]byte{}),
		func() error {
			panic("test panic")
		},
	}

	m, err := New(ctx)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	defer func() {
		if r := recover(); r != nil {
			t.Fatal("unexpected panic")
		}
	}()

	m.kill(conns)
}
