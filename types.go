package plex

import (
	"context"
	"io"
	"net"
)

type Killer interface {
	Kill() error
}

type Writer interface {
	io.WriteCloser
	Send(ctx context.Context, buffer int) (chan<- byte, error)
	Killer
}

type Reader interface {
	io.ReadCloser
	Recv(ctx context.Context, buffer int) (<-chan byte, error)
	Killer
}

type Connect func(context.Context, net.Addr) (net.Conn, error)

// Option is the function literate that is used to configure the multiplexer
// during initialization.
type Option func(*Multiplexer) error

type Conn interface {
	io.ReadWriteCloser
	RemoteAddr() net.Addr
}
