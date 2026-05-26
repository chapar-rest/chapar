//go:build js && wasm
// +build js,wasm

package idb

import (
	"context"
	"errors"
	"sync"
	"syscall/js"

	"github.com/hack-pad/safejs"
)

// Factory lets applications asynchronously access the indexed databases. A typical program will call Global() to access window.indexedDB.
type Factory struct {
	jsFactory safejs.Value
}

var (
	global     *Factory
	globalErr  error
	globalOnce sync.Once
)

// Global returns the global IndexedDB instance.
// Can be called multiple times, will always return the same result (or error if one occurs).
func Global() *Factory {
	globalOnce.Do(func() {
		var jsFactory safejs.Value
		jsFactory, globalErr = safejs.Global().Get("indexedDB")
		if globalErr != nil {
			return
		}
		var truthy bool
		truthy, globalErr = jsFactory.Truthy()
		if globalErr != nil {
			return
		}
		if truthy {
			global, globalErr = WrapFactory(safejs.Unsafe(jsFactory))
		} else {
			globalErr = errors.New("Global JS variable 'indexedDB' is not defined")
		}
	})
	if globalErr != nil {
		panic(globalErr)
	}
	return global
}

// WrapFactory wraps the given IDBFactory object
func WrapFactory(jsFactory js.Value) (*Factory, error) {
	return &Factory{
		jsFactory: safejs.Safe(jsFactory),
	}, nil
}

// Open requests to open a connection to a database.
func (f *Factory) Open(upgradeCtx context.Context, name string, version uint, upgrader Upgrader) (*OpenDBRequest, error) {
	args := []interface{}{name}
	if version > 0 {
		args = append(args, version)
	}
	reqValue, err := f.jsFactory.Call("open", args...)
	if err != nil {
		return nil, tryAsDOMException(err)
	}
	req := wrapRequest(nil, reqValue)
	return newOpenDBRequest(upgradeCtx, req, upgrader)
}

// DeleteDatabase requests the deletion of a database.
func (f *Factory) DeleteDatabase(name string) (*AckRequest, error) {
	reqValue, err := f.jsFactory.Call("deleteDatabase", name)
	if err != nil {
		return nil, tryAsDOMException(err)
	}
	req := wrapRequest(nil, reqValue)
	return newAckRequest(req), nil
}

// CompareKeys compares two keys and returns a result indicating which one is greater in value.
func (f *Factory) CompareKeys(a, b js.Value) (int, error) {
	compare, err := f.jsFactory.Call("cmp", a, b)
	if err != nil {
		return 0, tryAsDOMException(err)
	}
	return compare.Int()
}
