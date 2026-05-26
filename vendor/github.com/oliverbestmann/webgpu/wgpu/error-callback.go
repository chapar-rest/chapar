package wgpu

import "C"
import (
	"errors"
	"runtime"
	"strings"
)

/*
#include <wgpu.h>
*/
import "C"

import (
	"unsafe"
)

type errorCallback struct {
	handle
	callers [1]uintptr
	err     error
}

func acquireErrorCallback() *errorCallback {
	cb := allocErrorCallbackValue.Get()

	if cb.callers[0] != 0 {
		// someone else initialized this instance after calling Done()
		// This should never happen.
		panic("errorCallback missuse detected")
	}

	// get the location of the caller for error reporting
	runtime.Callers(2, cb.callers[:])

	if cb.handle == (handle{}) {
		// create a stable handle we can use to identify the callback
		// in c-land.
		cb.handle = newHandle(cb)

		// if the error callback gets cleaned up, we need to release
		// the c-handle too
		runtime.AddCleanup(cb, handle.Delete, cb.handle)
	}

	return cb
}

func (v *errorCallback) Done() {
	if v.callers[0] == 0 {
		// someone else called done on this instance.
		// This should never happen.
		panic("errorCallback missuse detected")
	}

	// use the callers field to indicate that the handle is initialized
	v.callers[0] = 0

	// return the instance back to the pool
	allocErrorCallbackValue.Put(v)
}

func (v *errorCallback) Handle(_ ErrorType, message string) {
	frames := runtime.CallersFrames(v.callers[:])
	frame, _ := frames.Next()

	context := frame.Func.Name()

	// strip github.com/.../ from method name
	if idx := strings.LastIndexByte(context, '/'); idx >= 0 {
		context = context[idx+1:]
	}

	v.err = Error{Context: context, Wrapped: errors.New(message)}
}

//export gowebgpu_error_callback_go
func gowebgpu_error_callback_go(_type C.WGPUErrorType, message C.WGPUStringView, userdata unsafe.Pointer) {
	handle := lookupHandle(userdata)
	ec, ok := handle.Value().(*errorCallback)
	if ok {
		ec.Handle(ErrorType(_type), C.GoStringN(message.data, C.int(message.length)))
	}
}
