//go:build js && wasm
// +build js,wasm

package idb

import (
	"context"
	"fmt"
	"log"

	"github.com/hack-pad/safejs"
)

// OpenDBRequest provides access to the results of requests to open or delete databases (performed using Factory.open and Factory.DeleteDatabase).
type OpenDBRequest struct {
	*Request
}

// Upgrader is a function that can upgrade the given database from an old version to a new one.
type Upgrader func(db *Database, oldVersion, newVersion uint) error

func newOpenDBRequest(ctx context.Context, req *Request, upgrader Upgrader) (*OpenDBRequest, error) {
	ctx, cancel := context.WithCancel(ctx)

	err := req.ListenSuccess(ctx, func() {
		defer cancel()
		err := openDBListenSuccess(req)
		if err != nil {
			panic(err)
		}
	})
	if err != nil {
		return nil, err
	}

	upgrade, err := safejs.FuncOf(func(this safejs.Value, args []safejs.Value) interface{} {
		err := openDBUpgradeNeeded(req, upgrader, args)
		if err != nil {
			panic(err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	_, err = req.jsRequest.Call(addEventListener, "upgradeneeded", upgrade)
	if err != nil {
		return nil, tryAsDOMException(err)
	}
	go func() {
		<-ctx.Done()
		_, err := req.jsRequest.Call(removeEventListener, "upgradeneeded", upgrade)
		if err != nil {
			panic(err)
		}
		upgrade.Release()
	}()
	return &OpenDBRequest{req}, nil
}

func openDBListenSuccess(req *Request) error {
	jsDB, err := req.result()
	if err != nil {
		return err
	}
	versionChange, err := safejs.FuncOf(func(safejs.Value, []safejs.Value) interface{} {
		log.Println("Version change detected, closing DB...")
		_, closeErr := jsDB.Call("close")
		if closeErr != nil {
			log.Println("Error closing DB:", closeErr)
		}
		return nil
	})
	if err != nil {
		return err
	}
	_, err = jsDB.Call(addEventListener, "versionchange", versionChange)
	return tryAsDOMException(err)
}

func openDBUpgradeNeeded(req *Request, upgrader Upgrader, args []safejs.Value) error {
	event := args[0]
	jsDatabase, err := req.result()
	if err != nil {
		return err
	}
	db := wrapDatabase(jsDatabase)
	oldVersionValue, err := event.Get("oldVersion")
	if err != nil {
		return err
	}
	oldVersion, err := oldVersionValue.Int()
	if err != nil {
		return err
	}
	newVersionValue, err := event.Get("newVersion")
	if err != nil {
		return err
	}
	newVersion, err := newVersionValue.Int()
	if err != nil {
		return err
	}
	if oldVersion < 0 || newVersion < 0 {
		return fmt.Errorf("Unexpected negative oldVersion or newVersion: %d, %d", oldVersion, newVersion)
	}
	return upgrader(db, uint(oldVersion), uint(newVersion))
}

// Result returns the result of the request. If the request failed and the result is not available, an error is returned.
func (o *OpenDBRequest) Result() (*Database, error) {
	db, err := o.Request.result()
	if err != nil {
		return nil, err
	}
	return wrapDatabase(db), nil
}

// Await waits for success or failure, then returns the results.
func (o *OpenDBRequest) Await(ctx context.Context) (*Database, error) {
	db, err := o.Request.await(ctx)
	if err != nil {
		return nil, err
	}
	return wrapDatabase(db), nil
}
