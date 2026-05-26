//go:build js && wasm

package safejs

import (
	"syscall/js"

	"github.com/hack-pad/safejs/internal/catch"
)

// CopyBytesToGo copies bytes from src to dst.
// Returns the number of bytes copied, which is the minimum of the lengths of src and dst.
// Returns an error if src is not an Uint8Array or Uint8ClampedArray.
func CopyBytesToGo(dst []byte, src Value) (int, error) {
	return catch.Try(func() int {
		return js.CopyBytesToGo(dst, src.jsValue)
	})
}

// CopyBytesToJS copies bytes from src to dst.
// Returns the number of bytes copied, which is the minimum of the lengths of src and dst.
// Returns an error if dst is not an Uint8Array or Uint8ClampedArray.
func CopyBytesToJS(dst Value, src []byte) (int, error) {
	return catch.Try(func() int {
		return js.CopyBytesToJS(dst.jsValue, src)
	})
}
