//go:build js && wasm

package safejs

import "syscall/js"

// Type represents the JavaScript type of a Value.
type Type int

// Available JavaScript types
const (
	TypeUndefined = Type(js.TypeUndefined)
	TypeNull      = Type(js.TypeNull)
	TypeBoolean   = Type(js.TypeBoolean)
	TypeNumber    = Type(js.TypeNumber)
	TypeString    = Type(js.TypeString)
	TypeSymbol    = Type(js.TypeSymbol)
	TypeObject    = Type(js.TypeObject)
	TypeFunction  = Type(js.TypeFunction)
)

func (t Type) String() string {
	// String() has a panic line, however it should be impossible to hit barring memory corruption
	return js.Type(t).String()
}
