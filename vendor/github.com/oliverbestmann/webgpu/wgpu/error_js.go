package wgpu

import (
	"errors"
	"fmt"
	"runtime"
	"strings"
)

func handleJsException(err *error) {
	r := recover()
	if r == nil {
		return
	}

	// get the callers.
	//  skip: this method, panic, js.Value.Call
	var callers [1]uintptr
	runtime.Callers(4, callers[:])

	frames := runtime.CallersFrames(callers[:])
	frame, _ := frames.Next()
	context := frame.Func.Name()

	// strip github.com/.../ from method name
	if idx := strings.LastIndexByte(context, '/'); idx >= 0 {
		context = context[idx+1:]
	}

	switch r := r.(type) {
	case error:
		*err = wrap(r, context)

	case fmt.Stringer:
		*err = Error{
			Context: context,
			Wrapped: errors.New(r.String()),
		}

	default:
		*err = Error{
			Context: context,
			Wrapped: fmt.Errorf("%v", r),
		}
	}
}
