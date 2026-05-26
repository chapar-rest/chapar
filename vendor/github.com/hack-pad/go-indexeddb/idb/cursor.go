//go:build js && wasm
// +build js,wasm

package idb

import (
	"syscall/js"

	"github.com/hack-pad/go-indexeddb/idb/internal/jscache"
	"github.com/hack-pad/safejs"
)

var (
	jsObjectStore        safejs.Value
	cursorDirectionCache jscache.Strings
)

func init() {
	var err error
	jsObjectStore, err = safejs.Global().Get("IDBObjectStore")
	if err != nil {
		panic(err)
	}
}

// CursorDirection is the direction of traversal of the cursor
type CursorDirection int

const (
	// CursorNext direction causes the cursor to be opened at the start of the source.
	CursorNext CursorDirection = iota
	// CursorNextUnique direction causes the cursor to be opened at the start of the source. For every key with duplicate values, only the first record is yielded.
	CursorNextUnique
	// CursorPrevious direction causes the cursor to be opened at the end of the source.
	CursorPrevious
	// CursorPreviousUnique direction causes the cursor to be opened at the end of the source. For every key with duplicate values, only the first record is yielded.
	CursorPreviousUnique
)

func parseCursorDirection(s string) CursorDirection {
	switch s {
	case "nextunique":
		return CursorNextUnique
	case "prev":
		return CursorPrevious
	case "prevunique":
		return CursorPreviousUnique
	default:
		return CursorNext
	}
}

func (d CursorDirection) String() string {
	switch d {
	case CursorNextUnique:
		return "nextunique"
	case CursorPrevious:
		return "prev"
	case CursorPreviousUnique:
		return "prevunique"
	default:
		return "next"
	}
}

func (d CursorDirection) jsValue() safejs.Value {
	return cursorDirectionCache.Value(d.String())
}

// Cursor represents a cursor for traversing or iterating over multiple records in a Database
type Cursor struct {
	txn      *Transaction
	jsCursor safejs.Value
	iterated bool // set to true when an iteration method is called, like Continue
}

func wrapCursor(txn *Transaction, jsCursor safejs.Value) *Cursor {
	if txn == nil {
		txn = (*Transaction)(nil)
	}
	return &Cursor{
		txn:      txn,
		jsCursor: jsCursor,
	}
}

// Source returns the ObjectStore or Index that the cursor is iterating
func (c *Cursor) Source() (objectStore *ObjectStore, index *Index, err error) {
	jsSource, err := c.jsCursor.Get("source")
	if err != nil {
		return
	}
	if isInstance, _ := jsSource.InstanceOf(jsObjectStore); isInstance {
		objectStore = wrapObjectStore(c.txn, jsSource)
	} else if isInstance, _ := jsSource.InstanceOf(jsIDBIndex); isInstance {
		index = wrapIndex(c.txn, jsSource)
	}
	return
}

// Direction returns the direction of traversal of the cursor
func (c *Cursor) Direction() (CursorDirection, error) {
	direction, err := c.jsCursor.Get("direction")
	if err != nil {
		return 0, err
	}
	directionStr, err := direction.String()
	return parseCursorDirection(directionStr), err
}

// Key returns the key for the record at the cursor's position. If the cursor is outside its range, this is set to undefined.
func (c *Cursor) Key() (js.Value, error) {
	value, err := c.jsCursor.Get("key")
	return safejs.Unsafe(value), err
}

// PrimaryKey returns the cursor's current effective primary key. If the cursor is currently being iterated or has iterated outside its range, this is set to undefined.
func (c *Cursor) PrimaryKey() (js.Value, error) {
	value, err := c.jsCursor.Get("primaryKey")
	return safejs.Unsafe(value), err
}

// Request returns the Request that was used to obtain the cursor.
func (c *Cursor) Request() (*Request, error) {
	reqValue, err := c.jsCursor.Get("request")
	if err != nil {
		return nil, err
	}
	return wrapRequest(c.txn, reqValue), nil
}

// Advance sets the number of times a cursor should move its position forward.
func (c *Cursor) Advance(count uint) error {
	c.iterated = true
	_, err := c.jsCursor.Call("advance", count)
	return tryAsDOMException(err)
}

// Continue advances the cursor to the next position along its direction.
func (c *Cursor) Continue() error {
	c.iterated = true
	_, err := c.jsCursor.Call("continue")
	return tryAsDOMException(err)
}

// ContinueKey advances the cursor to the next position along its direction.
func (c *Cursor) ContinueKey(key js.Value) error {
	c.iterated = true
	_, err := c.jsCursor.Call("continue", key)
	return tryAsDOMException(err)
}

// ContinuePrimaryKey sets the cursor to the given index key and primary key given as arguments. Returns an error if the source is not an index.
func (c *Cursor) ContinuePrimaryKey(key, primaryKey js.Value) error {
	c.iterated = true
	_, err := c.jsCursor.Call("continuePrimaryKey", key, primaryKey)
	return tryAsDOMException(err)
}

// Delete returns an AckRequest, and, in a separate thread, deletes the record at the cursor's position, without changing the cursor's position. This can be used to delete specific records.
func (c *Cursor) Delete() (*AckRequest, error) {
	reqValue, err := c.jsCursor.Call("delete")
	if err != nil {
		return nil, tryAsDOMException(err)
	}
	req := wrapRequest(c.txn, reqValue)
	return newAckRequest(req), nil
}

// Update returns a Request, and, in a separate thread, updates the value at the current position of the cursor in the object store. This can be used to update specific records.
func (c *Cursor) Update(value js.Value) (*Request, error) {
	reqValue, err := c.jsCursor.Call("update", value)
	if err != nil {
		return nil, tryAsDOMException(err)
	}
	return wrapRequest(c.txn, reqValue), nil
}

// CursorWithValue represents a cursor for traversing or iterating over multiple records in a database. It is the same as the Cursor, except that it includes the value property.
type CursorWithValue struct {
	*Cursor
}

func newCursorWithValue(cursor *Cursor) *CursorWithValue {
	return &CursorWithValue{cursor}
}

func wrapCursorWithValue(txn *Transaction, jsCursor safejs.Value) *CursorWithValue {
	return newCursorWithValue(wrapCursor(txn, jsCursor))
}

// Value returns the value of the current cursor
func (c *CursorWithValue) Value() (js.Value, error) {
	value, err := c.jsCursor.Get("value")
	return safejs.Unsafe(value), err
}
