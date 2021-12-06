package plex

import (
	"bytes"
	"context"
	"io"
	"testing"
)

func Test_multiplexer_queueWriters_canceled(t *testing.T) {
	for name, test := range ctxCancelTests() {
		t.Run(name, func(t *testing.T) {
			m := &multiplexer{ctx: test.parent}
			test.Eval(t, m.queueWriters(test.child, bytes.NewBuffer([]byte("test"))))
		})
	}
}

func Test_multiplexer_queueReaders_canceled(t *testing.T) {
	for name, test := range ctxCancelTests() {
		t.Run(name, func(t *testing.T) {
			m := &multiplexer{ctx: test.parent}
			test.Eval(t, m.queueReaders(test.child, bytes.NewBuffer([]byte("test"))))
		})
	}
}

func Test_multiplexer_queueReadWriters_canceled(t *testing.T) {
	for name, test := range ctxCancelTests() {
		t.Run(name, func(t *testing.T) {
			m := &multiplexer{ctx: test.parent}
			test.Eval(t, m.queueReadWriters(test.child, bytes.NewBuffer([]byte("test"))))
		})
	}
}

func Test_multiplexer_queueReadStreams_canceled(t *testing.T) {
	for name, test := range ctxCancelTests() {
		t.Run(name, func(t *testing.T) {
			m := &multiplexer{ctx: test.parent}
			test.Eval(t, m.queueReadStreams(test.child, &rStream{}))
		})
	}
}

func Test_multiplexer_queueWriteStreams_canceled(t *testing.T) {
	for name, test := range ctxCancelTests() {
		t.Run(name, func(t *testing.T) {
			m := &multiplexer{ctx: test.parent}
			test.Eval(t, m.queueWriteStreams(test.child, &wStream{}))
		})
	}
}

func Test_multiplexer_queue_canceled(t *testing.T) {
	for name, test := range ctxCancelTests() {
		t.Run(name, func(t *testing.T) {
			m := &multiplexer{ctx: test.parent}
			test.Eval(t, m.queue(test.child, &wStream{}))
		})
	}
}

type testreader struct {
	io.Reader
}

type testwriter struct {
	io.Writer
}

// nolint:funlen
func Test_multiplexer_queue(t *testing.T) {
	testdata := map[string]struct {
		inputs      []interface{}
		readercount int
		writercount int
		err         error
		hasErr      bool
	}{
		// TODO: Add Support for this
		"valid single stream": {
			inputs: []interface{}{&rwStream{
				ctx:    context.Background(),
				cancel: func() {}}},
			readercount: 0,
			writercount: 0,
			err:         nil,
			hasErr:      false,
		},
		// TODO: Add Support for this
		"valid stream slice; single stream": {
			inputs: []interface{}{[]Stream{&rwStream{
				ctx:    context.Background(),
				cancel: func() {},
			}}},
			readercount: 0,
			writercount: 0,
			err:         nil,
			hasErr:      false,
		},
		"valid single writestream": {
			inputs: []interface{}{&wStream{
				ctx:    context.Background(),
				cancel: func() {},
			}},
			writercount: 1,
			err:         nil,
			hasErr:      false,
		},
		"valid writestream slice; single writestream": {
			inputs: []interface{}{[]WriteStream{&wStream{
				ctx:    context.Background(),
				cancel: func() {},
			}}},
			writercount: 1,
			err:         nil,
			hasErr:      false,
		},
		"valid writestream slice; multiple writestream": {
			inputs: []interface{}{
				[]WriteStream{
					&wStream{
						ctx:    context.Background(),
						cancel: func() {},
					},
					&wStream{
						ctx:    context.Background(),
						cancel: func() {},
					},
					&wStream{
						ctx:    context.Background(),
						cancel: func() {},
					},
					&wStream{
						ctx:    context.Background(),
						cancel: func() {},
					},
					&wStream{
						ctx:    context.Background(),
						cancel: func() {},
					},
				}},
			writercount: 5,
			err:         nil,
			hasErr:      false,
		},
		"invalid writestream slice; single writestream, nil second": {
			inputs: []interface{}{
				[]WriteStream{
					&wStream{
						ctx:    context.Background(),
						cancel: func() {},
					},
					nil,
				}},
			writercount: 1,
			err:         ErrNil,
			hasErr:      true,
		},
		"invalid writestream slice; multi writestream, nil interleaved": {
			inputs: []interface{}{
				[]WriteStream{
					&wStream{
						ctx:    context.Background(),
						cancel: func() {},
					},
					nil,
					&wStream{
						ctx:    context.Background(),
						cancel: func() {},
					},
				}},
			writercount: 1,
			err:         ErrNil,
			hasErr:      true,
		},
		"valid single readstream": {
			inputs: []interface{}{&rStream{
				ctx:    context.Background(),
				cancel: func() {}}},
			readercount: 1,
			err:         nil,
			hasErr:      false,
		},
		"valid readstream slice; single readstream": {
			inputs: []interface{}{[]ReadStream{&rStream{
				ctx:    context.Background(),
				cancel: func() {},
			}}},
			readercount: 1,
			err:         nil,
			hasErr:      false,
		},
		"valid readstream slice; multiple readstream": {
			inputs: []interface{}{
				[]ReadStream{
					&rStream{
						ctx:    context.Background(),
						cancel: func() {},
					},
					&rStream{
						ctx:    context.Background(),
						cancel: func() {},
					},
					&rStream{
						ctx:    context.Background(),
						cancel: func() {},
					},
					&rStream{
						ctx:    context.Background(),
						cancel: func() {},
					},
					&rStream{
						ctx:    context.Background(),
						cancel: func() {},
					},
				}},
			readercount: 5,
			err:         nil,
			hasErr:      false,
		},
		"invalid readstream slice; single readstream, nil second": {
			inputs: []interface{}{
				[]ReadStream{
					&rStream{
						ctx:    context.Background(),
						cancel: func() {},
					},
					nil,
				}},
			readercount: 1,
			err:         ErrNil,
			hasErr:      true,
		},
		"invalid readstream slice; multi readstream, nil interleaved": {
			inputs: []interface{}{
				[]ReadStream{
					&rStream{
						ctx:    context.Background(),
						cancel: func() {},
					},
					nil,
					&rStream{
						ctx:    context.Background(),
						cancel: func() {},
					},
				}},
			readercount: 1,
			err:         ErrNil,
			hasErr:      true,
		},
		"valid single writer": {
			inputs: []interface{}{
				&testwriter{},
			},
			writercount: 1,
			err:         nil,
			hasErr:      false,
		},
		"valid writer slice; single writer": {
			inputs: []interface{}{
				[]io.Writer{
					&testwriter{},
				}},
			writercount: 1,
			err:         nil,
			hasErr:      false,
		},
		"valid writer slice; multiple writer": {
			inputs: []interface{}{
				[]io.Writer{
					&testwriter{},
					&testwriter{},
					&testwriter{},
					&testwriter{},
					&testwriter{},
				}},
			writercount: 5,
			err:         nil,
			hasErr:      false,
		},
		"invalid writer slice; single writer, nil second": {
			inputs: []interface{}{
				[]io.Writer{
					&testwriter{},
					nil,
				}},
			writercount: 1,
			err:         ErrNil,
			hasErr:      true,
		},
		"invalid writer slice; multi writer, nil interleaved": {
			inputs: []interface{}{
				[]io.Writer{
					&testwriter{},
					nil,
					&testwriter{},
				}},
			writercount: 1,
			err:         ErrNil,
			hasErr:      true,
		},
		"valid single reader": {
			inputs: []interface{}{
				&testreader{},
			},
			readercount: 1,
			err:         nil,
			hasErr:      false,
		},
		"valid reader slice; single reader": {
			inputs: []interface{}{
				[]io.Reader{
					&testreader{},
				}},
			readercount: 1,
			err:         nil,
			hasErr:      false,
		},
		"valid reader slice; multiple reader": {
			inputs: []interface{}{
				[]io.Reader{
					&testreader{},
					&testreader{},
					&testreader{},
					&testreader{},
					&testreader{},
				}},
			readercount: 5,
			err:         nil,
			hasErr:      false,
		},
		"invalid reader slice; single reader, nil second": {
			inputs: []interface{}{
				[]io.Reader{
					&testreader{},
					nil,
				}},
			readercount: 1,
			err:         ErrNil,
			hasErr:      true,
		},
		"invalid reader slice; multi reader, nil interleaved": {
			inputs: []interface{}{
				[]io.Reader{
					&testreader{},
					nil,
					&testreader{},
				}},
			readercount: 1,
			err:         ErrNil,
			hasErr:      true,
		},
		"valid single readwriter": {
			inputs:      []interface{}{bytes.NewBuffer([]byte{})},
			readercount: 1,
			writercount: 1,
			err:         nil,
			hasErr:      false,
		},
		"valid readwriter slice; single readwriter": {
			inputs: []interface{}{
				[]io.ReadWriter{
					bytes.NewBuffer([]byte{}),
				}},
			readercount: 1,
			writercount: 1,
			err:         nil,
			hasErr:      false,
		},
		"valid readwriter slice; multiple readwriter": {
			inputs: []interface{}{
				[]io.ReadWriter{
					bytes.NewBuffer([]byte{}),
					bytes.NewBuffer([]byte{}),
					bytes.NewBuffer([]byte{}),
					bytes.NewBuffer([]byte{}),
					bytes.NewBuffer([]byte{}),
				}},
			readercount: 5,
			writercount: 5,
			err:         nil,
			hasErr:      false,
		},
		"invalid readwriter slice; single readwriter, nil second": {
			inputs: []interface{}{
				[]io.ReadWriter{
					bytes.NewBuffer([]byte{}),
					nil,
				}},
			readercount: 1,
			writercount: 1,
			err:         ErrNil,
			hasErr:      true,
		},
		"invalid readwriter slice; multi readwriter, nil interleaved": {
			inputs: []interface{}{
				[]io.ReadWriter{
					bytes.NewBuffer([]byte{}),
					nil,
					bytes.NewBuffer([]byte{}),
				}},
			readercount: 1,
			writercount: 1,
			err:         ErrNil,
			hasErr:      true,
		},
		"invalid nil; toplevel": {
			inputs:      nil,
			readercount: 0,
			writercount: 0,
			err:         ErrNil,
			hasErr:      true,
		},
		"invalid nil; second level": {
			inputs:      []interface{}{nil},
			readercount: 0,
			writercount: 0,
			err:         ErrNil,
			hasErr:      true,
		},
		"invalid nil; third level": {
			inputs:      []interface{}{[]interface{}{nil}},
			readercount: 0,
			writercount: 0,
			err:         ErrNil,
			hasErr:      true,
		},
	}

	for name, test := range testdata {
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// Increased buffer to eval overflow
			bufferIncrease := 10

			m := &multiplexer{
				ctx:     ctx,
				cancel:  cancel,
				readers: make(chan ReadStream, test.readercount+bufferIncrease),
				writers: make(chan WriteStream, test.writercount+bufferIncrease),
			}
			err := m.queue(ctx, test.inputs...)

			if len(m.readers) != test.readercount {
				t.Fatalf("unexpected reader count: %d", len(m.readers))
			}

			if len(m.writers) != test.writercount {
				t.Fatalf("unexpected writer count: %d", len(m.writers))
			}

			if err != nil {
				if !test.hasErr {
					t.Fatalf("unexpected error: %v", err)
				}

				if test.err != nil && test.err != err {
					t.Fatalf("unexpected error: %v", err)
				}

				return
			}
		})
	}
}
