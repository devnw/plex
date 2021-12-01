package plex

import (
	"bytes"
	"context"
	"crypto/sha1"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"testing"
)

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

func Test_recoverErr(t *testing.T) {
	testdata := map[string]struct {
		value      interface{}
		underlying error
		expected   error
	}{
		"nil": {
			value:    nil,
			expected: nil,
		},
		"string": {
			value:    "test error",
			expected: errors.New("test error"),
		},
		"error": {
			value:    errors.New("test error"),
			expected: errors.New("test error"),
		},
		"recover type proxy": {
			value:    365,
			expected: errors.New("panic: 365"),
		},
		"nil w/ underlying": {
			value:      nil,
			expected:   errors.New("underlying"),
			underlying: errors.New("underlying"),
		},
		"string w/ underlying": {
			value:      "test error",
			expected:   errors.New("test error"),
			underlying: errors.New("underlying"),
		},
		"error w/ underlying": {
			value:      errors.New("test error"),
			expected:   errors.New("test error"),
			underlying: errors.New("underlying"),
		},
		"recover type proxy w/ underlying": {
			value:      365,
			expected:   errors.New("panic: 365"),
			underlying: errors.New("underlying"),
		},
	}

	for name, test := range testdata {
		t.Run(name, func(t *testing.T) {
			err := recoverErr(test.underlying, test.value)
			if err == nil && test.expected == nil {
				return
			}

			if err.Error() != test.expected.Error() {
				t.Errorf(
					"expected %s, got %s",
					test.expected.Error(),
					err.Error(),
				)
			}

			under, ok := err.(wrappedError)
			if !ok {
				if test.underlying != nil && test.value != nil {
					t.Fatalf(
						"expected %s, got %s",
						test.underlying.Error(),
						err.Error(),
					)
				}

				return
			}

			if under.Unwrap() != test.underlying {
				t.Fatalf(
					"expected %s, got %s",
					test.underlying.Error(),
					under.Unwrap().Error(),
				)
			}
		})
	}
}

func Test_ctx(t *testing.T) {
	testdata := map[string]struct {
		ctx context.Context
	}{
		"non nil": {
			context.Background(),
		},
		"nil": {
			nil,
		},
	}

	for name, test := range testdata {
		t.Run(name, func(t *testing.T) {
			ctx, cancel := _ctx(test.ctx)
			if ctx == nil {
				t.Fatal("expected non nil context")
			}

			if cancel == nil {
				t.Fatal("expected non nil cancel")
			}
		})
	}
}

func Test_merge(t *testing.T) {
	testdata := map[string]struct {
		parent context.Context
		child  context.Context
	}{
		"non-nil": {
			parent: context.Background(),
			child:  context.TODO(),
		},
		"nil": {
			parent: context.Background(),
			child:  nil,
		},
	}

	for name, test := range testdata {
		t.Run(name, func(t *testing.T) {
			ctx := merge(test.parent, test.child)
			if ctx == nil {
				t.Fatal("expected non nil context")
			}

			if test.child != nil && ctx != test.child {
				t.Fatal("expected child context")
			}
		})
	}
}
