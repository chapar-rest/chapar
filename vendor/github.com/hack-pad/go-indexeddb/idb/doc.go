//go:build js && wasm
// +build js,wasm

/*
Package idb is a low-level driver that provides type-safe bindings to IndexedDB in Wasm programs.
The primary focus is to align with the IndexedDB spec, followed by ease of use.

To get started, get the global indexedDB instance with idb.Global(). See below for examples.
*/
package idb
