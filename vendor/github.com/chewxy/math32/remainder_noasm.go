//go:build noasm || tinygo || (!amd64 && !s390x && !arm && !ppc64le && !386 && !wasm)
// +build noasm tinygo !amd64,!s390x,!arm,!ppc64le,!386,!wasm

package math32

const haveArchRemainder = false

func archRemainder(x, y float32) float32 {
	panic("not implemented")
}
