//go:build js && wasm

package safejs

import (
	"syscall/js"

	"github.com/hack-pad/safejs/internal/catch"
)

// Error wraps a JavaScript error.
type Error struct {
	err js.Error
}

// Error implements the error interface.
func (e Error) Error() string {
	errStr, err := catch.Try(e.err.Error)
	if err != nil {
		return "failed generating error message: " + err.Error()
	}
	return errStr
}
