//go:build js && wasm
// +build js,wasm

package idb

import (
	"syscall/js"

	"github.com/hack-pad/safejs"
)

// ObjectStoreOptions contains all available options for creating an ObjectStore
type ObjectStoreOptions struct {
	KeyPath       js.Value
	AutoIncrement bool
}

// ObjectStore represents an object store in a database. Records within an object store are sorted according to their keys. This sorting enables fast insertion, look-up, and ordered retrieval.
type ObjectStore struct {
	base *baseObjectStore // don't embed to avoid generated docs with the wrong receiver type (ObjectStore vs *ObjectStore)
}

func wrapObjectStore(txn *Transaction, jsObjectStore safejs.Value) *ObjectStore {
	return &ObjectStore{wrapBaseObjectStore(txn, jsObjectStore)}
}

// IndexNames returns a list of the names of indexes on objects in this object store.
func (o *ObjectStore) IndexNames() ([]string, error) {
	indexNames, err := o.base.jsObjectStore.Get("indexNames")
	if err != nil {
		return nil, err
	}
	return stringsFromArray(indexNames)
}

// KeyPath returns the key path of this object store. If this returns js.Null(), the application must provide a key for each modification operation.
func (o *ObjectStore) KeyPath() (js.Value, error) {
	keyPath, err := o.base.jsObjectStore.Get("keyPath")
	return safejs.Unsafe(keyPath), err
}

// Name returns the name of this object store.
func (o *ObjectStore) Name() (string, error) {
	name, err := o.base.jsObjectStore.Get("name")
	if err != nil {
		return "", err
	}
	return name.String()
}

// Transaction returns the Transaction object to which this object store belongs.
func (o *ObjectStore) Transaction() (*Transaction, error) {
	if o.base.txn == (*Transaction)(nil) {
		return nil, errNotInTransaction
	}
	return o.base.txn, nil
}

// AutoIncrement returns the value of the auto increment flag for this object store.
func (o *ObjectStore) AutoIncrement() (bool, error) {
	autoIncrement, err := o.base.jsObjectStore.Get("autoIncrement")
	if err != nil {
		return false, err
	}
	return autoIncrement.Bool()
}

// Add returns an AckRequest, and, in a separate thread, creates a structured clone of the value, and stores the cloned value in the object store. This is for adding new records to an object store.
func (o *ObjectStore) Add(value js.Value) (*AckRequest, error) {
	reqValue, err := o.base.jsObjectStore.Call("add", value)
	if err != nil {
		return nil, tryAsDOMException(err)
	}
	req := wrapRequest(o.base.txn, reqValue)
	return newAckRequest(req), nil
}

// AddKey is the same as Add, but includes the key to use to identify the record.
func (o *ObjectStore) AddKey(key, value js.Value) (*AckRequest, error) {
	reqValue, err := o.base.jsObjectStore.Call("add", value, key)
	if err != nil {
		return nil, tryAsDOMException(err)
	}
	req := wrapRequest(o.base.txn, reqValue)
	return newAckRequest(req), nil
}

// Clear returns an AckRequest, then clears this object store in a separate thread. This is for deleting all current records out of an object store.
func (o *ObjectStore) Clear() (*AckRequest, error) {
	reqValue, err := o.base.jsObjectStore.Call("clear")
	if err != nil {
		return nil, tryAsDOMException(err)
	}
	req := wrapRequest(o.base.txn, reqValue)
	return newAckRequest(req), nil
}

// Count returns a UintRequest, and, in a separate thread, returns the total number of records in the store.
func (o *ObjectStore) Count() (*UintRequest, error) {
	return o.base.Count()
}

// CountKey returns a UintRequest, and, in a separate thread, returns the total number of records that match the provided key.
func (o *ObjectStore) CountKey(key js.Value) (*UintRequest, error) {
	return o.base.CountKey(safejs.Safe(key))
}

// CountRange returns a UintRequest, and, in a separate thread, returns the total number of records that match the provided KeyRange.
func (o *ObjectStore) CountRange(keyRange *KeyRange) (*UintRequest, error) {
	return o.base.CountRange(keyRange)
}

// CreateIndex creates a new index during a version upgrade, returning a new Index object in the connected database.
func (o *ObjectStore) CreateIndex(name string, keyPath js.Value, options IndexOptions) (*Index, error) {
	jsIndex, err := o.base.jsObjectStore.Call("createIndex", name, keyPath, map[string]interface{}{
		"unique":     options.Unique,
		"multiEntry": options.MultiEntry,
	})
	if err != nil {
		return nil, tryAsDOMException(err)
	}
	return wrapIndex(o.base.txn, jsIndex), nil
}

// Delete returns an AckRequest, and, in a separate thread, deletes the store object selected by the specified key. This is for deleting individual records out of an object store.
func (o *ObjectStore) Delete(key js.Value) (*AckRequest, error) {
	reqValue, err := o.base.jsObjectStore.Call("delete", key)
	if err != nil {
		return nil, tryAsDOMException(err)
	}
	req := wrapRequest(o.base.txn, reqValue)
	return newAckRequest(req), nil
}

// DeleteIndex destroys the specified index in the connected database, used during a version upgrade.
func (o *ObjectStore) DeleteIndex(name string) error {
	_, err := o.base.jsObjectStore.Call("deleteIndex", name)
	return tryAsDOMException(err)
}

// GetAllKeys returns an ArrayRequest that retrieves record keys for all objects in the object store.
func (o *ObjectStore) GetAllKeys() (*ArrayRequest, error) {
	return o.base.GetAllKeys()
}

// GetAllKeysRange returns an ArrayRequest that retrieves record keys for all objects in the object store matching the specified query. If maxCount is 0, retrieves all objects matching the query.
func (o *ObjectStore) GetAllKeysRange(query *KeyRange, maxCount uint) (*ArrayRequest, error) {
	return o.base.GetAllKeysRange(query, maxCount)
}

// Get returns a Request, and, in a separate thread, returns the objects selected by the specified key. This is for retrieving specific records from an object store.
func (o *ObjectStore) Get(key js.Value) (*Request, error) {
	return o.base.Get(safejs.Safe(key))
}

// GetKey returns a Request, and, in a separate thread retrieves and returns the record key for the object matching the specified parameter.
func (o *ObjectStore) GetKey(value js.Value) (*Request, error) {
	return o.base.GetKey(safejs.Safe(value))
}

// Index opens an index from this object store after which it can, for example, be used to return a sequence of records sorted by that index using a cursor.
func (o *ObjectStore) Index(name string) (*Index, error) {
	jsIndex, err := o.base.jsObjectStore.Call("index", name)
	if err != nil {
		return nil, tryAsDOMException(err)
	}
	return wrapIndex(o.base.txn, jsIndex), nil
}

// Put returns a Request, and, in a separate thread, creates a structured clone of the value, and stores the cloned value in the object store. This is for updating existing records in an object store when the transaction's mode is readwrite.
func (o *ObjectStore) Put(value js.Value) (*Request, error) {
	reqValue, err := o.base.jsObjectStore.Call("put", value)
	if err != nil {
		return nil, tryAsDOMException(err)
	}
	return wrapRequest(o.base.txn, reqValue), nil
}

// PutKey is the same as Put, but includes the key to use to identify the record.
func (o *ObjectStore) PutKey(key, value js.Value) (*Request, error) {
	reqValue, err := o.base.jsObjectStore.Call("put", value, key)
	if err != nil {
		return nil, tryAsDOMException(err)
	}
	return wrapRequest(o.base.txn, reqValue), nil
}

// OpenCursor returns a CursorWithValueRequest, and, in a separate thread, returns a new CursorWithValue. Used for iterating through an object store by primary key with a cursor.
func (o *ObjectStore) OpenCursor(direction CursorDirection) (*CursorWithValueRequest, error) {
	return o.base.OpenCursor(direction)
}

// OpenCursorKey is the same as OpenCursor, but opens a cursor over the given key instead.
func (o *ObjectStore) OpenCursorKey(key js.Value, direction CursorDirection) (*CursorWithValueRequest, error) {
	return o.base.OpenCursorKey(safejs.Safe(key), direction)
}

// OpenCursorRange is the same as OpenCursor, but opens a cursor over the given range instead.
func (o *ObjectStore) OpenCursorRange(keyRange *KeyRange, direction CursorDirection) (*CursorWithValueRequest, error) {
	return o.base.OpenCursorRange(keyRange, direction)
}

// OpenKeyCursor returns a CursorRequest, and, in a separate thread, returns a new Cursor. Used for iterating through all keys in an object store.
func (o *ObjectStore) OpenKeyCursor(direction CursorDirection) (*CursorRequest, error) {
	return o.base.OpenKeyCursor(direction)
}

// OpenKeyCursorKey is the same as OpenKeyCursor, but opens a cursor over the given key instead.
func (o *ObjectStore) OpenKeyCursorKey(key js.Value, direction CursorDirection) (*CursorRequest, error) {
	return o.base.OpenKeyCursorKey(safejs.Safe(key), direction)
}

// OpenKeyCursorRange is the same as OpenKeyCursor, but opens a cursor over the given key range instead.
func (o *ObjectStore) OpenKeyCursorRange(keyRange *KeyRange, direction CursorDirection) (*CursorRequest, error) {
	return o.base.OpenKeyCursorRange(keyRange, direction)
}
