//go:build js

// Package jsx provides essential JavaScript functions that are used
// widely in wgpu and are very useful for an wasm / js application.
package jsx

import (
	"log/slog"
	"syscall/js"
	"unsafe"
)

// BytesToJS converts the given bytes to a js Uint8ClampedArray
// by using the global wasm memory bytes. This avoids the
// copying present in [js.CopyBytesToJS].
func BytesToJS(b []byte) js.Value {
	ptr := uintptr(unsafe.Pointer(&b[0]))
	// We directly pass the offset and length to the constructor to avoid calling subarray or slice,
	// thereby improving performance and safety (this fixes a detached array buffer crash).
	return js.Global().Get("Uint8ClampedArray").New(js.Global().Get("wasm").Get("instance").Get("exports").Get("mem").Get("buffer"), ptr, len(b))
}

// Await is a helper function equivalent to await in JS.
// It is copied from https://go-review.googlesource.com/c/go/+/150917/
func Await(promise js.Value) (result js.Value, ok bool) {
	if promise.Type() != js.TypeObject || promise.Get("then").Type() != js.TypeFunction {
		return promise, true
	}

	done := make(chan struct{})

	onResolve := js.FuncOf(func(this js.Value, args []js.Value) any {
		result = args[0]
		ok = true
		close(done)
		return nil
	})
	defer onResolve.Release()

	onReject := js.FuncOf(func(this js.Value, args []js.Value) any {
		result = args[0]
		ok = false
		slog.Error("wgpu.AwaitJS: promise rejected", "reason", result)
		close(done)
		return nil
	})
	defer onReject.Release()

	promise.Call("then", onResolve, onReject)
	<-done
	return
}
