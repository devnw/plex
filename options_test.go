package plex

import (
	"context"
	"errors"
	"fmt"
	"net"
	"testing"
	"time"
)

type conntest struct {
	conns    []net.Conn
	err      bool
	expected int
}

func connectionTests() map[string]conntest {
	return map[string]conntest{
		"single": {
			[]net.Conn{
				&testconn{},
			},
			false,
			1,
		},
		"single nil": {
			[]net.Conn{
				nil,
			},
			false,
			0,
		},
		"single valid, single nil (first)": {
			[]net.Conn{
				nil,
				&testconn{},
			},
			false,
			1,
		},
		"single valid, single nil (last)": {
			[]net.Conn{
				&testconn{},
				nil,
			},
			false,
			1,
		},
		"multi": {
			[]net.Conn{
				&testconn{},
				&testconn{},
				&testconn{},
				&testconn{},
				&testconn{},
			},
			false,
			5,
		},
		"multi nil": {
			[]net.Conn{
				&testconn{},
				nil,
				&testconn{},
				nil,
				&testconn{},
			},
			false,
			3,
		},
		"multi valid, single nil": {
			[]net.Conn{
				&testconn{},
				&testconn{},
				nil,
				&testconn{},
				&testconn{},
			},
			false,
			4,
		},
		"multi different remote": {
			[]net.Conn{
				&testconn{
					addr: "127.0.0.1",
				},
				&testconn{},
			},
			true,
			0,
		},
	}
}

func Test_WithConnections(t *testing.T) {
	for name, test := range connectionTests() {
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			m, err := New(ctx, WithConnections(test.conns...))
			if err != nil {
				if !test.err {
					t.Fatal(err)
				}
				return
			}

			defer func() {
				err := m.Close()
				if err != nil {
					t.Errorf("Close() failed: %v", err)
				}
			}()

			if len(m.readers) != test.expected {
				t.Fatalf("expected %d readers, got %d", test.expected, len(m.readers))
			}

			if len(m.writers) != test.expected {
				t.Fatalf("expected %d writers, got %d", test.expected, len(m.writers))
			}
		})
	}
}

func Test_WithConnector(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fn := func(ctx context.Context, addr net.Addr) (net.Conn, error) {
		return &testconn{
			addr: "0.0.0.0",
			tcp:  true,
		}, nil
	}

	m, err := New(
		ctx,
		WithConnector(fn),
	)

	if err != nil {
		t.Fatal(err)
	}

	if m.connector == nil {
		t.Fatal("expected connector to be set")
	}
}

func Test_WithConnector_nil(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_, err := New(
		ctx,
		WithConnector(nil),
	)

	if err != errConnectorNil {
		t.Fatalf("expected ErrConnectorNil; got %v", err)
	}
}

func WithErrOption() Option {
	return func(m *Multiplexer) error {
		return errors.New("err")
	}
}

func Test_OptionError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_, err := New(
		ctx,
		WithErrOption(),
	)

	if err == nil {
		t.Fatal("expected error")
	}
}

func Test_WithMaxCapacity(t *testing.T) {
	testdata := map[int]error{
		1:  nil,
		5:  nil,
		10: nil,
		15: nil,
		-1: errInvalidMaxCapacity,
	}

	for capacity, expErr := range testdata {
		t.Run(fmt.Sprintf("cap-%v", capacity), func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			m, err := New(
				ctx,
				WithMaxCapacity(capacity),
			)

			if err != expErr {
				t.Fatalf("expected %v; got %v", expErr, err)
			}

			if err != nil {
				return
			}

			if m.capacity != capacity {
				t.Fatalf("expected capacity %v; got %v", capacity, m.capacity)
			}

			if cap(m.readers) != capacity {
				t.Fatalf(
					"expected reader capacity %v; got %v",
					capacity,
					cap(m.readers),
				)
			}

			if cap(m.writers) != capacity {
				t.Fatalf(
					"expected writer capacity %v; got %v",
					capacity,
					cap(m.writers),
				)
			}
		})
	}
}

func Test_WithMaxCapacity_MoreConnections(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_, err := New(
		ctx,
		WithMaxCapacity(1),
		WithConnections(
			&testconn{},
			&testconn{},
		),
	)

	if err != errTooManyConns {
		t.Fatalf("expected %v; got %v", errTooManyConns, err)
	}
}

func Test_WithAutoScale_NoConnect(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_, err := New(
		ctx,
		WithAutoScaling(time.Millisecond),
	)

	if err != errImproperAutoScalingNilConnector {
		t.Fatalf(
			"expected %v; got %v",
			errImproperAutoScalingNilConnector,
			err,
		)
	}
}

func Test_WithAutoScaling(t *testing.T) {
	testdata := map[time.Duration]error{
		time.Nanosecond:   nil,
		time.Millisecond:  nil,
		time.Second:       nil,
		time.Minute:       nil,
		time.Duration(-1): errInvalidTimeout,
	}

	for duration, expErr := range testdata {
		t.Run(fmt.Sprintf("scale_timeout-%s", duration), func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			m, err := New(
				ctx,
				WithAutoScaling(duration),
				WithConnector(
					func(context.Context, net.Addr) (net.Conn, error) {
						return &testconn{}, nil
					}),
			)

			if err != expErr {
				t.Fatalf("expected %v; got %v", expErr, err)
			}

			if err != nil {
				return
			}

			if m.scaleTimeout == nil {
				t.Fatal("expected scaleTimeout to be set")
			}

			if *m.scaleTimeout != duration {
				t.Fatalf("expected capacity %v; got %v", duration, *m.scaleTimeout)
			}
		})
	}
}
