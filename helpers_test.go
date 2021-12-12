package plex

import (
	"context"
	"errors"
	"testing"
)

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
