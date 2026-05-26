//go:build js && wasm

package safejs

import (
	"fmt"
	"syscall/js"
)

// Global returns the JavaScript global object, usually "window" or "global".
func Global() Value {
	return Safe(js.Global())
}

// MustGetGlobal fetches the given global, then verifies it is truthy. Panics on error or falsy values.
// This is intended for simple global variable initialization, like preparing classes for later instantiation.
//
// For example:
//
//	var jsUint8Array = safejs.MustGetGlobal("Uint8Array")
func MustGetGlobal(property string) Value {
	value, err := getGlobal(property)
	if err != nil {
		panic(err)
	}
	return value
}

func getGlobal(property string) (Value, error) {
	value, err := Global().Get(property)
	if err != nil {
		return Value{}, err
	}
	truthy, err := value.Truthy()
	if err != nil {
		return Value{}, err
	}
	if !truthy {
		return Value{}, fmt.Errorf("global %q is not defined", property)
	}
	return value, nil
}
