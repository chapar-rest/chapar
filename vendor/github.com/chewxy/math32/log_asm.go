// Copyright 2021 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !noasm && !tinygo && (amd64 || arm || s390x || 386 || arm64 || ppc64le || wasm)
// +build !noasm
// +build !tinygo
// +build amd64 arm s390x 386 arm64 ppc64le wasm

package math32

const haveArchLog = true

func archLog(x float32) float32
