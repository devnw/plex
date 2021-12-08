package plex

import (
	"net"
	"testing"
)

func Test_ErrConnection(t *testing.T) {
	testdata := map[string]net.Addr{
		"local": &testaddr{
			tcp:  true,
			addr: "127.0.0.1:8080",
		},
		"nil": nil,
	}

	for name, test := range testdata {
		t.Run(name, func(t *testing.T) {
			err := disconnected(test)
			if err == nil {
				t.Error("expected error")
			}

			if err.Error() != "disconnected" {
				t.Error("expected error message to be 'disconnected'")
			}

			e, ok := err.(*ErrConnection)
			if !ok {
				t.Error("expected error to be of type ErrConnection")
			}

			if e.Addr != test {
				t.Error("expected error to have Addr set to test")
			}
		})
	}
}

func Test_ErrAddrMismatch(t *testing.T) {
	e := &errAddrMismatch{
		Expected: &testaddr{
			tcp:  true,
			addr: "127.0.0.1",
		},
		Actual: &testaddr{
			tcp:  true,
			addr: "0.0.0.0",
		},
	}

	err := error(e)

	expected := "address mismatch; expected: tcp:127.0.0.1; actual: tcp:0.0.0.0"

	if err.Error() != expected {
		t.Errorf(
			"expected error message to be %q; got %q",
			expected,
			err.Error(),
		)
	}
}
