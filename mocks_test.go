package plex

import (
	"bytes"
	"context"
	"crypto/sha1"
	"fmt"
	"io"
	"math/rand"
	"net"
	"sync"
	"testing"
	"time"
)

type testconn struct {
	rwc  *rwStream
	tcp  bool
	addr string
}

func (c *testconn) Read(b []byte) (n int, err error) {
	if c.rwc == nil {
		return 0, io.EOF
	}

	return c.rwc.Read(b)
}

func (c *testconn) Write(b []byte) (n int, err error) {
	if c.rwc == nil {
		return 0, io.EOF
	}

	return c.rwc.Write(b)
}

func (c *testconn) Close() error {
	return nil
}

func (c *testconn) RemoteAddr() net.Addr {
	return &testaddr{
		tcp:  c.tcp,
		addr: c.addr,
	}
}

// Not implemented for now, just here to make bconn
// look like a net.Conn
func (*testconn) LocalAddr() net.Addr                { return nil }
func (*testconn) SetDeadline(t time.Time) error      { return nil }
func (*testconn) SetReadDeadline(t time.Time) error  { return nil }
func (*testconn) SetWriteDeadline(t time.Time) error { return nil }

type iotestconn struct {
	rw   io.ReadWriter
	tcp  bool
	addr string
}

func (c *iotestconn) Read(b []byte) (n int, err error) {
	if c.rw == nil {
		return 0, io.EOF
	}

	return c.rw.Read(b)
}

func (c *iotestconn) Write(b []byte) (n int, err error) {
	if c.rw == nil {
		return 0, io.EOF
	}

	return c.rw.Write(b)
}

func (c *iotestconn) Close() error {
	return nil
}

func (c *iotestconn) RemoteAddr() net.Addr {
	return &testaddr{
		tcp:  c.tcp,
		addr: c.addr,
	}
}

// Not implemented for now, just here to make bconn
// look like a net.Conn
func (*iotestconn) LocalAddr() net.Addr                { return nil }
func (*iotestconn) SetDeadline(t time.Time) error      { return nil }
func (*iotestconn) SetReadDeadline(t time.Time) error  { return nil }
func (*iotestconn) SetWriteDeadline(t time.Time) error { return nil }

type testaddr struct {
	tcp  bool
	addr string
}

func (t *testaddr) Network() string {
	if t.tcp {
		return "tcp"
	}
	return "udp"
}

func (t *testaddr) String() string {
	return t.addr
}

type ctxtest struct {
	parent context.Context
	child  context.Context
}

func ctxCancelTests() map[string]ctxtest {
	ctx := context.Background()

	cancelledCtx, cancelledCancel := context.WithCancel(context.Background())
	cancelledCancel()

	return map[string]ctxtest{
		"child cancel": {
			parent: ctx,
			child:  cancelledCtx,
		},
		"parent cancel": {
			parent: cancelledCtx,
			child:  ctx,
		},
		"parent cancel, nil child": {
			parent: cancelledCtx,
			child:  nil,
		},
	}
}

func (test ctxtest) Eval(t *testing.T, err error) {
	if err == nil {
		t.Fatalf("expected error")
	}

	if err != context.Canceled {
		t.Fatalf("expected context canceled; got %v", err)
	}

	if test.parent != nil {
		if test.parent.Err() == err {
			return
		}
	}

	if test.child != nil {
		if test.child.Err() == err {
			return
		}
	}

	t.Fatalf("expected error to match parent or child")
}

// SumByteSlice is a map with a key which is the sha1sum of the byte slice
type SumByteSlice map[string][]byte

func (s SumByteSlice) SliceOfReadWriter() []io.ReadWriter {
	rws := make([]io.ReadWriter, 0, len(s))
	for _, v := range s {
		rws = append(rws, bytes.NewBuffer(v))
	}

	return rws
}

// setOfRandBytes returns a set of random byte slices of the given size
func setOfRandBytes(size int) (data SumByteSlice, err error) {
	data = SumByteSlice{}

	for i := 0; i < size; i++ {
		// Mod 1024 to make sure the random byteData
		// are not too large
		byteData, err := randBytes(rand.Int() % 1024)
		if err != nil {
			return nil, err
		}

		data[fmt.Sprintf("%x", sha1.Sum(byteData))] = byteData
	}

	return data, nil
}

//	randBytes returns a random byte slice of the given size
func randBytes(size int) ([]byte, error) {
	buff := make([]byte, size)
	n, err := rand.Read(buff)
	if err != nil {
		return nil, fmt.Errorf("error reading random data: %s", err)
	}

	return buff[:n], nil
}

type wrappedError interface {
	error
	Unwrap() error
}

func newStream(ctx context.Context) *testconn {
	ctx, cancel := _ctx(ctx)

	return &testconn{
		&rwStream{
			ctx:    ctx,
			cancel: cancel,
			data:   make(chan byte),
		},
		true,
		"0.0.0.0",
	}
}

type rwStream struct {
	ctx    context.Context
	cancel context.CancelFunc
	data   chan byte
	mu     sync.Mutex
	wg     sync.WaitGroup
}

func (rws *rwStream) Recv(ctx context.Context, buffer int) (<-chan byte, error) {
	ctx = merge(rws.ctx, ctx)
	out := make(chan byte)

	rws.mu.Lock()
	defer rws.mu.Unlock()
	rws.wg.Add(1)

	go func(out chan<- byte, in <-chan byte) {
		defer func() {
			_ = recover() // TODO: handle in the future?
		}()
		defer rws.wg.Done()
		defer close(out)

		for {
			select {
			case <-rws.ctx.Done():
				return
			case <-ctx.Done():
				return
			case b, ok := <-in:
				if !ok {
					return
				}

				select {
				case <-rws.ctx.Done():
					return
				case <-ctx.Done():
					return
				case out <- b:
				}
			}
		}
	}(out, rws.data)

	return out, nil
}

func (rws *rwStream) Send(ctx context.Context, buffer int) (chan<- byte, error) {
	ctx = merge(rws.ctx, ctx)
	in := make(chan byte)

	rws.mu.Lock()
	defer rws.mu.Unlock()
	rws.wg.Add(1)

	go func(internal chan<- byte, in <-chan byte) {
		defer func() {
			_ = recover() // TODO: handle in the future?
		}()
		defer rws.wg.Done()

		for {
			select {
			case <-rws.ctx.Done():
				return
			case <-ctx.Done():
				return
			case b, ok := <-in:
				if !ok {
					return
				}

				select {
				case <-rws.ctx.Done():
					return
				case <-ctx.Done():
					return
				case internal <- b:
				}
			}
		}
	}(rws.data, in)

	return in, nil
}

func (rws *rwStream) Read(p []byte) (n int, err error) {
	var i int
	for i = 0; i < len(p); i++ {
		select {
		case <-rws.ctx.Done():
			return 0, rws.ctx.Err()
		case b, ok := <-rws.data:
			if !ok {
				return 0, io.EOF
			}

			p[i] = b
		}
	}

	return i, nil
}

func (rws *rwStream) Write(p []byte) (n int, err error) {
	var i int
	for i = 0; i < len(p); i++ {
		select {
		case <-rws.ctx.Done():
			return 0, rws.ctx.Err()
		case rws.data <- p[i]:
		}
	}

	return i, nil
}

func (rws *rwStream) Close() error {
	defer func() {
		_ = recover() // TODO: handle in the future?
	}()

	rws.cancel()
	<-rws.ctx.Done()

	rws.mu.Lock()
	defer rws.mu.Unlock()

	rws.wg.Wait()

	return nil
}
