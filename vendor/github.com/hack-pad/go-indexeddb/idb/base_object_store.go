//go:build js && wasm
// +build js,wasm

package idb

import (
	"github.com/hack-pad/safejs"
)

// baseObjectStore is the common implementation for both object stores and indexes.
type baseObjectStore struct {
	txn           *Transaction
	jsObjectStore safejs.Value
}

func wrapBaseObjectStore(txn *Transaction, jsObjectStore safejs.Value) *baseObjectStore {
	if txn == nil {
		txn = (*Transaction)(nil)
	}
	return &baseObjectStore{
		txn:           txn,
		jsObjectStore: jsObjectStore,
	}
}

// Count returns a UintRequest, and, in a separate thread, returns the total number of records in the store or index.
func (b *baseObjectStore) Count() (*UintRequest, error) {
	reqValue, err := b.jsObjectStore.Call("count")
	if err != nil {
		return nil, tryAsDOMException(err)
	}
	req := wrapRequest(b.txn, reqValue)
	return newUintRequest(req), nil
}

// CountKey returns a UintRequest, and, in a separate thread, returns the total number of records that match the provided key.
func (b *baseObjectStore) CountKey(key safejs.Value) (*UintRequest, error) {
	reqValue, err := b.jsObjectStore.Call("count", key)
	if err != nil {
		return nil, tryAsDOMException(err)
	}
	req := wrapRequest(b.txn, reqValue)
	return newUintRequest(req), nil
}

// CountRange returns a UintRequest, and, in a separate thread, returns the total number of records that match the provided KeyRange.
func (b *baseObjectStore) CountRange(keyRange *KeyRange) (*UintRequest, error) {
	reqValue, err := b.jsObjectStore.Call("count", keyRange.jsKeyRange)
	if err != nil {
		return nil, tryAsDOMException(err)
	}
	req := wrapRequest(b.txn, reqValue)
	return newUintRequest(req), nil
}

// GetAllKeys returns an ArrayRequest that retrieves record keys for all objects in the object store or index.
func (b *baseObjectStore) GetAllKeys() (*ArrayRequest, error) {
	reqValue, err := b.jsObjectStore.Call("getAllKeys")
	if err != nil {
		return nil, tryAsDOMException(err)
	}
	req := wrapRequest(b.txn, reqValue)
	return newArrayRequest(req), nil
}

// GetAllKeysRange returns an ArrayRequest that retrieves record keys for all objects in the object store or index matching the specified query. If maxCount is 0, retrieves all objects matching the query.
func (b *baseObjectStore) GetAllKeysRange(query *KeyRange, maxCount uint) (*ArrayRequest, error) {
	args := []interface{}{query.jsKeyRange}
	if maxCount > 0 {
		args = append(args, maxCount)
	}
	reqValue, err := b.jsObjectStore.Call("getAllKeys", args...)
	if err != nil {
		return nil, tryAsDOMException(err)
	}
	req := wrapRequest(b.txn, reqValue)
	return newArrayRequest(req), nil
}

// Get returns a Request, and, in a separate thread, returns the objects selected by the specified key. This is for retrieving specific records from an object store or index.
func (b *baseObjectStore) Get(key safejs.Value) (*Request, error) {
	reqValue, err := b.jsObjectStore.Call("get", key)
	if err != nil {
		return nil, tryAsDOMException(err)
	}
	return wrapRequest(b.txn, reqValue), nil
}

// GetKey returns a Request, and, in a separate thread retrieves and returns the record key for the object matching the specified parameter.
func (b *baseObjectStore) GetKey(value safejs.Value) (*Request, error) {
	reqValue, err := b.jsObjectStore.Call("getKey", value)
	if err != nil {
		return nil, tryAsDOMException(err)
	}
	return wrapRequest(b.txn, reqValue), nil
}

// OpenCursor returns a CursorWithValueRequest, and, in a separate thread, returns a new CursorWithValue. Used for iterating through an object store or index by primary key with a cursor.
func (b *baseObjectStore) OpenCursor(direction CursorDirection) (*CursorWithValueRequest, error) {
	reqValue, err := b.jsObjectStore.Call("openCursor", safejs.Null(), direction.jsValue())
	if err != nil {
		return nil, tryAsDOMException(err)
	}
	req := wrapRequest(b.txn, reqValue)
	return newCursorWithValueRequest(req), nil
}

// OpenCursorKey is the same as OpenCursor, but opens a cursor over the given key instead.
func (b *baseObjectStore) OpenCursorKey(key safejs.Value, direction CursorDirection) (*CursorWithValueRequest, error) {
	reqValue, err := b.jsObjectStore.Call("openCursor", key, direction.jsValue())
	if err != nil {
		return nil, tryAsDOMException(err)
	}
	req := wrapRequest(b.txn, reqValue)
	return newCursorWithValueRequest(req), nil
}

// OpenCursorRange is the same as OpenCursor, but opens a cursor over the given range instead.
func (b *baseObjectStore) OpenCursorRange(keyRange *KeyRange, direction CursorDirection) (*CursorWithValueRequest, error) {
	reqValue, err := b.jsObjectStore.Call("openCursor", keyRange.jsKeyRange, direction.jsValue())
	if err != nil {
		return nil, tryAsDOMException(err)
	}
	req := wrapRequest(b.txn, reqValue)
	return newCursorWithValueRequest(req), nil
}

// OpenKeyCursor returns a CursorRequest, and, in a separate thread, returns a new Cursor. Used for iterating through all keys in an object store or index.
func (b *baseObjectStore) OpenKeyCursor(direction CursorDirection) (*CursorRequest, error) {
	reqValue, err := b.jsObjectStore.Call("openKeyCursor", safejs.Null(), direction.jsValue())
	if err != nil {
		return nil, tryAsDOMException(err)
	}
	req := wrapRequest(b.txn, reqValue)
	return newCursorRequest(req), nil
}

// OpenKeyCursorKey is the same as OpenKeyCursor, but opens a cursor over the given key instead.
func (b *baseObjectStore) OpenKeyCursorKey(key safejs.Value, direction CursorDirection) (*CursorRequest, error) {
	reqValue, err := b.jsObjectStore.Call("openKeyCursor", key, direction.jsValue())
	if err != nil {
		return nil, tryAsDOMException(err)
	}
	req := wrapRequest(b.txn, reqValue)
	return newCursorRequest(req), nil
}

// OpenKeyCursorRange is the same as OpenKeyCursor, but opens a cursor over the given key range instead.
func (b *baseObjectStore) OpenKeyCursorRange(keyRange *KeyRange, direction CursorDirection) (*CursorRequest, error) {
	reqValue, err := b.jsObjectStore.Call("openKeyCursor", keyRange.jsKeyRange, direction.jsValue())
	if err != nil {
		return nil, tryAsDOMException(err)
	}
	req := wrapRequest(b.txn, reqValue)
	return newCursorRequest(req), nil
}
