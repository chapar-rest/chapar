//go:build wasm
// +build wasm

// Package jswrapper contains a Wrapper for interoperating with JavaScript.
package jswrapper

import "syscall/js"

// Wrapper is implemented by types that are backed by a JavaScript value.
// This wrapper was previously included in the standard library.
type Wrapper interface {
	// JSValue returns a JavaScript value associated with an object.
	JSValue() js.Value
}
