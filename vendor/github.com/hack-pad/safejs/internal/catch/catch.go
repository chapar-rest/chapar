//go:build js && wasm

// Package catch runs functions and returns panic values as errors instead.
package catch

import (
	"fmt"
	"syscall/js"

	"github.com/hack-pad/safejs/internal/stackerr"
)

// Try runs fn and returns the result. If fn panicked, the panic value is returned as an error instead.
func Try[Result any](fn func() Result) (result Result, err error) {
	defer recoverErr(&err)
	result = fn()
	return
}

// TrySideEffect is like Try, but does not have a return value.
func TrySideEffect(fn func()) (err error) {
	defer recoverErr(&err)
	fn()
	return
}

func recoverErr(err *error) {
	value := recover()
	valueErr := recoverValueToError(value)
	if valueErr != nil {
		*err = stackerr.WithStack(valueErr)
	}
}

func recoverValueToError(value any) error {
	if value == nil {
		return nil
	}
	switch value := value.(type) {
	case error:
		return value
	case js.Value:
		return js.Error{Value: value}
	default:
		return fmt.Errorf("%+v", value)
	}
}
