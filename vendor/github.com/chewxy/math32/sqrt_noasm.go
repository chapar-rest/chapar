// Copyright 2021 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build noasm || tinygo || (!386 && !amd64 && !arm64 && !arm && !ppc64le && !s390x && !wasm)
// +build noasm tinygo !386,!amd64,!arm64,!arm,!ppc64le,!s390x,!wasm

package math32

const haveArchSqrt = false

func archSqrt(x float32) float32 {
	panic("not implemented")
}
