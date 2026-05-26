//go:build js && wasm

/*
Package safejs provides guardrails around the [syscall/js] package, like turning thrown exceptions into errors.

Since [syscall/js] is experimental, this package may have breaking changes to stay aligned with the latest versions of Go.
*/
package safejs
