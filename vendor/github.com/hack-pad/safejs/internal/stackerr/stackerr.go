//go:build js && wasm

// Package stackerr adds stack traces to verbose error messages.
package stackerr

import (
	"fmt"
	"runtime"
	"strings"
)

type stackError struct {
	err   error
	stack *stacktrace
}

// WithStack returns 'err' with a stack trace in its verbose formatter output.
// Returns nil if err is nil.
func WithStack(err error) error {
	if err == nil {
		return nil
	}

	return &stackError{
		err:   err,
		stack: collectStacktrace(3),
	}
}

func (s *stackError) Error() string {
	return s.err.Error()
}

func (s *stackError) Unwrap() error {
	return s.err
}

func (s *stackError) Format(f fmt.State, verb rune) {
	switch verb {
	case 'v':
		if f.Flag('+') {
			fmt.Fprintf(f, "%+v\n%s", s.err, s.stack)
			return
		}
		fmt.Fprint(f, s.Error())
	case 's':
		fmt.Fprint(f, s.Error())
	case 'q':
		fmt.Fprintf(f, "%q", s.Error())
	}
}

type stacktrace struct {
	callers []uintptr
}

func collectStacktrace(skip int) *stacktrace {
	const (
		maxFrames = 32
	)
	pc := make([]uintptr, maxFrames)
	n := runtime.Callers(1+skip, pc)
	return &stacktrace{
		callers: pc[:n],
	}
}

func (s *stacktrace) String() string {
	var sb strings.Builder
	frames := runtime.CallersFrames(s.callers)
	for frame, next := frames.Next(); next; frame, next = frames.Next() {
		funcName := "unknown"
		if frame.Func != nil {
			funcName = frame.Func.Name()
		}
		sb.WriteString(fmt.Sprintf("%s\n\t%s:%d\n", funcName, frame.File, frame.Line))
	}
	return sb.String()
}
