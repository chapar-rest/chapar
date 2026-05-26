//go:build js && wasm
// +build js,wasm

package idb

import (
	"github.com/hack-pad/go-indexeddb/idb/internal/jscache"
	"github.com/hack-pad/safejs"
)

// Database provides a connection to a database. You can use a Database object to open a transaction on your database then create, manipulate, and delete objects (data) in that database.
type Database struct {
	jsDB        safejs.Value
	callStrings jscache.Strings
}

func wrapDatabase(jsDB safejs.Value) *Database {
	return &Database{jsDB: jsDB}
}

// Name returns the name of the connected database.
func (db *Database) Name() (string, error) {
	value, err := db.jsDB.Get("name")
	if err != nil {
		return "", err
	}
	return value.String()
}

// Version returns the version of the connected database.
func (db *Database) Version() (uint, error) {
	value, err := db.jsDB.Get("version")
	if err != nil {
		return 0, err
	}
	intValue, err := value.Int()
	return uint(intValue), err
}

// ObjectStoreNames returns a list of the names of the object stores currently in the connected database.
func (db *Database) ObjectStoreNames() ([]string, error) {
	array, err := db.jsDB.Get("objectStoreNames")
	if err != nil {
		return nil, err
	}
	return stringsFromArray(array)
}

// CreateObjectStore creates and returns a new object store or index.
func (db *Database) CreateObjectStore(name string, options ObjectStoreOptions) (*ObjectStore, error) {
	jsObjectStore, err := db.jsDB.Call("createObjectStore", name, map[string]interface{}{
		"autoIncrement": options.AutoIncrement,
		"keyPath":       options.KeyPath,
	})
	if err != nil {
		return nil, tryAsDOMException(err)
	}
	return wrapObjectStore(nil, jsObjectStore), nil
}

// DeleteObjectStore destroys the object store with the given name in the connected database, along with any indexes that reference it.
func (db *Database) DeleteObjectStore(name string) error {
	_, err := db.jsDB.Call("deleteObjectStore", name)
	return tryAsDOMException(err)
}

// Close closes the connection to a database.
func (db *Database) Close() error {
	_, err := db.jsDB.Call("close")
	return tryAsDOMException(err)
}

// Transaction returns a transaction object containing the Transaction.ObjectStore() method, which you can use to access your object store.
func (db *Database) Transaction(mode TransactionMode, objectStoreName string, objectStoreNames ...string) (_ *Transaction, err error) {
	return db.TransactionWithOptions(TransactionOptions{Mode: mode}, objectStoreName, objectStoreNames...)
}

// TransactionOptions contains all available options for creating and starting a Transaction
type TransactionOptions struct {
	Mode       TransactionMode
	Durability TransactionDurability
}

// TransactionWithOptions returns a transaction object containing the Transaction.ObjectStore() method, which you can use to access your object store.
func (db *Database) TransactionWithOptions(options TransactionOptions, objectStoreName string, objectStoreNames ...string) (*Transaction, error) {
	objectStoreNames = append([]string{objectStoreName}, objectStoreNames...) // require at least one name

	optionsMap := make(map[string]interface{})
	if options.Durability != DurabilityDefault {
		optionsMap["durability"] = options.Durability.jsValue()
	}

	args := []interface{}{sliceFromStrings(objectStoreNames), options.Mode.jsValue()}
	if len(optionsMap) > 0 {
		args = append(args, optionsMap)
	}

	jsTxn, err := db.jsDB.Call("transaction", args...)
	if err != nil {
		return nil, tryAsDOMException(err)
	}
	return wrapTransaction(db, jsTxn), nil
}
