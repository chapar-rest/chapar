//go:build js && wasm

package safejs

import (
	"syscall/js"

	"github.com/hack-pad/safejs/internal/catch"
)

// Func is a wrapped Go function to be called by JavaScript.
type Func struct {
	fn js.Func
}

// FuncOf returns a function to be used by JavaScript. See [js.FuncOf] for details.
func FuncOf(fn func(this Value, args []Value) any) (Func, error) {
	jsFunc, err := toJSFunc(fn)
	return Func{
		fn: jsFunc,
	}, err
}

func toJSFunc(fn func(this Value, args []Value) any) (js.Func, error) {
	jsFunc := func(this js.Value, args []js.Value) any {
		result := fn(Safe(this), toValues(args))
		return toJSValue(result)
	}
	return catch.Try(func() js.Func {
		return js.FuncOf(jsFunc)
	})
}

// Release frees up resources allocated for the function. The function must not be invoked after calling Release.
// It is allowed to call Release while the function is still running.
func (f Func) Release() {
	f.fn.Release()
}

// Value returns this Func's inner Value. For example, using value.Invoke() calls the function.
//
// Equivalent to accessing [js.Func]'s embedded [js.Value] field, only as a safejs type.
func (f Func) Value() Value {
	return Safe(f.fn.Value)
}
