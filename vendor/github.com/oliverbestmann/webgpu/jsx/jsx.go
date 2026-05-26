//go:build js

// Package jsx provides essential JavaScript functions that are used
// widely in wgpu and are very useful for an wasm / js application.
package jsx

import (
	"syscall/js"
)

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
		close(done)
		return nil
	})
	defer onReject.Release()

	promise.Call("then", onResolve, onReject)
	<-done
	return
}
