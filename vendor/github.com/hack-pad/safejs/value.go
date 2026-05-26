//go:build js && wasm

package safejs

import (
	"fmt"
	"syscall/js"

	"github.com/hack-pad/safejs/internal/catch"
)

// Value is a safer version of js.Value. Any panic returns an error instead.
type Value struct {
	jsValue js.Value
}

// Safe wraps a js.Value into a safejs.Value.
// Ideal for use in libraries where exposed types must match the standard library.
func Safe(value js.Value) Value {
	return Value{
		jsValue: value,
	}
}

// Unsafe unwraps a safejs.Value back into its js.Value.
// Ideal for use in libraries where exposed types must match the standard library.
func Unsafe(value Value) js.Value {
	return value.jsValue
}

// Null returns the JavaScript value of "null".
func Null() Value {
	return Safe(js.Null())
}

// Undefined returns the JavaScript value of "undefined".
func Undefined() Value {
	return Safe(js.Undefined())
}

func toJSValue(jsValue any) any {
	switch value := jsValue.(type) {
	case Value:
		return value.jsValue
	case Func:
		return value.fn
	case Error:
		return value.err
	case map[string]any:
		newValue := make(map[string]any)
		for mapKey, mapValue := range value {
			newValue[mapKey] = toJSValue(mapValue)
		}
		return newValue
	case []any:
		newValue := make([]any, len(value))
		for i, arg := range value {
			newValue[i] = toJSValue(arg)
		}
		return newValue
	default:
		return jsValue
	}
}

func toJSValues(args []any) []any {
	return toJSValue(args).([]any)
}

func toValues(args []js.Value) []Value {
	newArgs := make([]Value, len(args))
	for i, arg := range args {
		newArgs[i] = Safe(arg)
	}
	return newArgs
}

// ValueOf returns value as a JavaScript value. See [js.ValueOf] for details.
func ValueOf(value any) (Value, error) {
	jsValue, err := catch.Try(func() js.Value {
		return js.ValueOf(value)
	})
	return Safe(jsValue), err
}

// Bool attempts to convert this value into a boolean, otherwise returns an error.
func (v Value) Bool() (bool, error) {
	return catch.Try(v.jsValue.Bool)
}

// Call does a JavaScript call to the method m of value v with the given arguments.
// The arguments are mapped to JavaScript values according to the ValueOf function.
// Returns an error if v has no method m, the arguments failed to map to JavaScript values, or the function throws an error.
func (v Value) Call(m string, args ...any) (Value, error) {
	args = toJSValues(args)
	return catch.Try(func() Value {
		return Safe(v.jsValue.Call(m, args...))
	})
}

// Delete deletes the JavaScript property p of value v. Returns an error if v is not a JavaScript object.
func (v Value) Delete(p string) error {
	return catch.TrySideEffect(func() {
		v.jsValue.Delete(p)
	})
}

// Equal reports whether v and w are equal according to JavaScript's === operator.
func (v Value) Equal(w Value) bool {
	return v.jsValue.Equal(w.jsValue)
}

// Float returns the value v as a float64. Returns an error if v is not a JavaScript number.
func (v Value) Float() (float64, error) {
	return catch.Try(v.jsValue.Float)
}

// Get returns the JavaScript property p of value v. Returns an error if v is not a JavaScript object.
func (v Value) Get(p string) (Value, error) {
	return catch.Try(func() Value {
		return Safe(v.jsValue.Get(p))
	})
}

// Index returns JavaScript index i of value v. Returns an error if v is not a JavaScript object.
func (v Value) Index(i int) (Value, error) {
	return catch.Try(func() Value {
		return Safe(v.jsValue.Index(i))
	})
}

// InstanceOf reports whether v is an instance of type t according to JavaScript's instanceof operator.
// Returns an error if v is not a constructable type.
func (v Value) InstanceOf(t Value) (bool, error) {
	// Type failures in JS throw "TypeError: Right-hand side of 'instanceof' is not an object"
	// so catch those cases here.
	//
	// A valid type is a function with a field "prototype" which is an object.
	if t.Type() != TypeFunction {
		return false, fmt.Errorf("invalid type for instanceof: %v", t.Type())
	}
	prototype, err := t.Get("prototype")
	if err != nil {
		return false, fmt.Errorf("invalid constructor type for instanceof: %v", err)
	} else if prototype.Type() != TypeObject {
		return false, fmt.Errorf("invalid constructor type for instanceof: %v", prototype.Type())
	}
	return catch.Try(func() bool {
		return v.jsValue.InstanceOf(t.jsValue)
	})
}

// Int returns the value v truncated to an int. Returns an error if v is not a JavaScript number.
func (v Value) Int() (int, error) {
	return catch.Try(v.jsValue.Int)
}

// Invoke does a JavaScript call of the value v with the given arguments.
// The arguments get mapped to JavaScript values according to the ValueOf function.
// Returns an error if v is not a JavaScript function, the arguments failed to map to JavaScript values, or the function throws an error.
func (v Value) Invoke(args ...any) (Value, error) {
	args = toJSValues(args)
	return catch.Try(func() Value {
		return Safe(v.jsValue.Invoke(args...))
	})
}

// IsNaN reports whether v is the JavaScript value "NaN".
func (v Value) IsNaN() bool {
	return v.jsValue.IsNaN()
}

// IsNull reports whether v is the JavaScript value "null".
func (v Value) IsNull() bool {
	return v.jsValue.IsNull()
}

// IsUndefined reports whether v is the JavaScript value "undefined".
func (v Value) IsUndefined() bool {
	return v.jsValue.IsUndefined()
}

// Length returns the JavaScript property "length" of v.
// Returns an error if v is not a JavaScript object.
func (v Value) Length() (int, error) {
	return catch.Try(v.jsValue.Length)
}

// New uses JavaScript's "new" operator with value v as constructor and the given arguments.
// The arguments get mapped to JavaScript values according to the ValueOf function.
// Returns an error if v is not a JavaScript function, the arguments failed to map to JavaScript values, or the constructor throws an error.
func (v Value) New(args ...any) (Value, error) {
	args = toJSValues(args)
	return catch.Try(func() Value {
		return Safe(v.jsValue.New(args...))
	})
}

// Set sets the JavaScript property p of value v to ValueOf(x).
// Returns an error if v is not a JavaScript object or x failed to map to a JavaScript value.
func (v Value) Set(p string, x any) error {
	x = toJSValue(x)
	return catch.TrySideEffect(func() {
		v.jsValue.Set(p, x)
	})
}

// SetIndex sets the JavaScript index i of value v to ValueOf(x).
// Returns an error if if v is not a JavaScript object or x failed to map to a JavaScript value.
func (v Value) SetIndex(i int, x any) error {
	x = toJSValue(x)
	return catch.TrySideEffect(func() {
		v.jsValue.SetIndex(i, x)
	})
}

// String returns the value v as a string.
// Unlike the other getters, String() does not return an error if v's Type is not TypeString.
// Instead, it returns a string of the form "<T>" or "<T: V>" where T is v's type and V is a string representation of v's value.
//
// Returns an error if v is an invalid type or the string failed to load from the JavaScript runtime.
//
// NOTE: [syscall/js] takes the stance that String is a special case due to Go's String method convention and avoids panicking.
// However, js.String() can still fail in other ways so an error is returned anyway.
func (v Value) String() (string, error) {
	return catch.Try(v.jsValue.String)
}

// Truthy returns the JavaScript "truthiness" of the value v.
// In JavaScript, false, 0, "", null, undefined, and NaN are "falsy", and everything else is "truthy".
// See https://developer.mozilla.org/en-US/docs/Glossary/Truthy.
//
// Returns an error if v's type is invalid or if the value fails to load from the JavaScript runtime.
func (v Value) Truthy() (bool, error) {
	return catch.Try(v.jsValue.Truthy)
}

// Type returns the JavaScript type of the value v.
// It is similar to JavaScript's typeof operator, except it returns TypeNull instead of TypeObject for null.
func (v Value) Type() Type {
	return Type(v.jsValue.Type())
}
