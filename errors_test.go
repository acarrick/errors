package errors

import (
	"encoding"
	"io"
	"reflect"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	tests := []struct {
		msg    string
		opts   []interface{}
		expect string
	}{
		{
			msg:    "",
			opts:   nil,
			expect: "",
		},
		{
			msg:    "xx",
			opts:   nil,
			expect: "xx",
		},
		{
			msg: "xx",
			opts: []interface{}{
				"key1", "val1",
				"key2", 2,
			},
			expect: "xx key1=val1 key2=2",
		},
		{
			msg: "msg",
			opts: []interface{}{
				"key", time.Time{},
			},
			expect: "msg key=0001-01-01 00:00:00 +0000 UTC",
		},
	}

	for _, tt := range tests {
		got := New(tt.msg).With(tt.opts...)
		if got.Error() != tt.expect {
			t.Errorf("New.Error(): got: %q, want %q", got, tt.expect)
		}
	}
}

func TestWrapNil(t *testing.T) {
	got := Wrap(nil, "no error")
	if got != nil {
		t.Errorf("Wrap(nil, \"no error\"): got %#v, expected nil", got)
	}
}

type nilError struct{}

func (nilError) Error() string { return "nil error" }

func TestCause(t *testing.T) {
	x := New("error")
	tests := []struct {
		err  error
		want error
	}{{
		// nil error is nil
		err:  nil,
		want: nil,
	}, {
		// explicit nil error is nil
		err:  (error)(nil),
		want: nil,
	}, {
		// typed nil is nil
		err:  (*nilError)(nil),
		want: (*nilError)(nil),
	}, {
		// uncaused error is unaffected
		err:  io.EOF,
		want: io.EOF,
	}, {
		// caused error returns cause
		err:  Wrap(io.EOF, "ignored"),
		want: io.EOF,
	}, {
		err:  x, // return from errors.New
		want: x,
	}}

	for i, tt := range tests {
		got := Cause(tt.err)
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("test %d: got %#v, want %#v", i+1, got, tt.want)
		}
	}
}

func TestAttachNil(t *testing.T) {
	got := Wrap(nil, "")
	if got != nil {
		t.Errorf("Attach(nil, \"no error\"): got %#v, expected nil", got)
	}
}

func TestAttach(t *testing.T) {
	tests := []struct {
		cause         error
		opts          []interface{}
		expectedMsg   string
		expectedCause error
		expectedErr   error
	}{
		{
			// this test case tests that when no options are passed, the
			// original error is returned
			cause:         io.EOF,
			opts:          nil,
			expectedMsg:   "EOF",
			expectedCause: io.EOF,
			expectedErr:   io.EOF,
		},
		{
			cause:         io.EOF,
			opts:          []interface{}{"k1", "v1", "k2", "v2"},
			expectedMsg:   "EOF k1=v1 k2=v2",
			expectedCause: io.EOF,
		},
		{
			cause:         Wrap(io.EOF, "something failed").With("k3", "v3"),
			opts:          []interface{}{"k1", "v1", "k2", "v2"},
			expectedMsg:   "something failed k3=v3 k1=v1 k2=v2: EOF",
			expectedCause: io.EOF,
		},
	}

	for i, tt := range tests {
		err := Wrap(tt.cause, "").With(tt.opts...)
		actualMsg := err.Error()
		if actualMsg != tt.expectedMsg {
			t.Errorf("%d: expected=%q, actual=%q", i, tt.expectedMsg, actualMsg)
		}
		actualCause := Cause(err)
		if actualCause != tt.expectedCause {
			t.Errorf("%d: cause: expected=%v, actual=%v", i, tt.expectedCause, actualCause)
		}

		// only test if non-nil in the test case
		if tt.expectedErr != nil {
			if tt.expectedErr.Error() != err.Error() {
				t.Errorf("%d: error: expected=%v, actual=%v", i, tt.expectedErr, err)
			}
		}
	}
}

func TestKeyvals(t *testing.T) {
	tests := []struct {
		err        error
		errKeyvals []interface{}
		ctx        Context
		ctxKeyvals []interface{}
	}{
		{
			err:        New("message"),
			errKeyvals: []interface{}{"msg", "message"},
		},
		{
			err:        New("message").With("k1", "v1", "k2", 2),
			errKeyvals: []interface{}{"msg", "message", "k1", "v1", "k2", 2},
		},
		{
			err:        Wrap(io.EOF, "message"),
			errKeyvals: []interface{}{"msg", "message", "cause", "EOF"},
		},
		{
			err:        Wrap(io.EOF, "message").With(),
			errKeyvals: []interface{}{"msg", "message", "cause", "EOF"},
		},
		{
			err:        Wrap(io.EOF, "message").With("k1", "v1", "k2", 2),
			errKeyvals: []interface{}{"msg", "message", "k1", "v1", "k2", 2, "cause", "EOF"},
		},
		{
			err:        Wrap(io.EOF, "").With("k1", "v1", "k2", 2),
			errKeyvals: []interface{}{"msg", "EOF", "k1", "v1", "k2", 2},
		},
		{
			ctx:        With(),
			ctxKeyvals: nil,
		},
		{
			ctx:        With("k1", "v1", "k2", 2),
			ctxKeyvals: []interface{}{"k1", "v1", "k2", 2},
		},
		{
			ctx:        With("k1", "v1", "k2", 2).With("k3", 3),
			ctxKeyvals: []interface{}{"k1", "v1", "k2", 2, "k3", 3},
		},
	}

	type keyvalser interface {
		Keyvals() []interface{}
	}

	for i, tt := range tests {
		if tt.err != nil {
			keyvals, ok := tt.err.(keyvalser)
			if !ok {
				t.Errorf("%d: expected Keyvals(), none available", i)
				continue
			}
			kvs := keyvals.Keyvals()
			if !reflect.DeepEqual(tt.errKeyvals, kvs) {
				t.Errorf("%d: expected %v, actual %v", i, tt.errKeyvals, kvs)
			}
		}
		if tt.ctx != nil {
			keyvals, ok := tt.ctx.(keyvalser)
			if !ok {
				t.Errorf("%d: expected Keyvals(), none available", i)
				continue
			}
			kvs := keyvals.Keyvals()
			if !reflect.DeepEqual(tt.ctxKeyvals, kvs) {
				t.Errorf("%d: expected %v, actual %v", i, tt.ctxKeyvals, kvs)
			}
		}
	}
}

func TestMarshalText(t *testing.T) {
	tests := []struct {
		err  error
		text string
	}{
		{
			err:  New("error message"),
			text: "error message",
		},
		{
			err:  Wrap(io.EOF, "error message"),
			text: "error message: EOF",
		},
		{
			err:  Wrap(io.EOF, ""),
			text: "EOF",
		},
		{
			err:  New("error message").With("a", 1),
			text: "error message a=1",
		},
		{
			err:  Wrap(io.EOF, "error message").With("b2", "b2"),
			text: "error message b2=b2: EOF",
		},
		{
			err:  Wrap(io.EOF, "").With("c3", 3),
			text: "EOF c3=3",
		},
	}
	for i, tt := range tests {
		m := tt.err.(encoding.TextMarshaler)
		b, err := m.MarshalText()
		if err != nil {
			t.Errorf("%d: want no error, got %v", i, err)
			continue
		}
		if want, got := tt.text, string(b); want != got {
			t.Errorf("%d: want %q, got %q", i, want, got)
		}
	}
}
