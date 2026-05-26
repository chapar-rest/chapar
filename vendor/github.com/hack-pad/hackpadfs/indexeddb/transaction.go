//go:build wasm
// +build wasm

package indexeddb

import (
	"context"
	"sort"
	"sync"
	"sync/atomic"

	"github.com/hack-pad/go-indexeddb/idb"
	"github.com/hack-pad/hackpadfs"
	"github.com/hack-pad/hackpadfs/keyvalue"
	"github.com/hack-pad/hackpadfs/keyvalue/blob"
)

type transaction struct {
	ctx            context.Context
	abort          context.CancelFunc
	store          *store
	txn            *idb.Transaction
	results        map[keyvalue.OpID]keyvalue.OpResult
	pendingResults []func()
	nextOp         keyvalue.OpID
	resultsMu      sync.Mutex
}

func (t *transaction) newOp() keyvalue.OpID {
	nextOp := atomic.AddInt64((*int64)(&t.nextOp), 1)
	return keyvalue.OpID(nextOp - 1)
}

func (t *transaction) setResult(op keyvalue.OpID, result keyvalue.OpResult) {
	t.resultsMu.Lock()
	t.results[op] = result
	t.resultsMu.Unlock()
}

func (t *transaction) setPendingResult(op keyvalue.OpID, req *getFileRequest) {
	t.resultsMu.Lock()
	t.pendingResults = append(t.pendingResults, func() {
		result := keyvalue.OpResult{Op: op}
		if req != nil {
			result.Record, result.Err = req.Result()
		}
		t.setResult(op, result)
	})
	t.resultsMu.Unlock()
}

func (t *transaction) setPendingValidateErr(op keyvalue.OpID, parentExistsReq *parentDirExistsReq) {
	if parentExistsReq == nil {
		return
	}
	t.resultsMu.Lock()
	t.pendingResults = append(t.pendingResults, func() {
		err := parentExistsReq.Err()
		t.setResult(op, keyvalue.OpResult{Op: op, Err: err})
	})
	t.resultsMu.Unlock()
}

func (t *transaction) Get(path string) (op keyvalue.OpID) {
	op = t.newOp()
	infos, err := t.txn.ObjectStore(infoStore)
	if err != nil {
		t.setResult(op, keyvalue.OpResult{Op: op, Err: err})
		return
	}
	req, err := t.store.getFile(infos, path)
	if err != nil {
		t.setResult(op, keyvalue.OpResult{Op: op, Err: err})
		return
	}
	t.setPendingResult(op, req)
	return
}

func (t *transaction) GetHandler(path string, handler keyvalue.OpHandler) (op keyvalue.OpID) {
	op = t.newOp()
	infos, err := t.txn.ObjectStore(infoStore)
	if err != nil {
		t.setResult(op, keyvalue.OpResult{Op: op, Err: err})
		return
	}
	req, err := t.store.getFile(infos, path)
	if err != nil {
		t.setResult(op, keyvalue.OpResult{Op: op, Err: err})
		return
	}
	listenErr := req.Listen(t.ctx, func() {
		record, err := req.Result()
		result := keyvalue.OpResult{
			Op:     op,
			Record: record,
			Err:    err,
		}
		if err := handler.Handle(t, result); err != nil {
			result.Err = err
		}
		t.setResult(op, result)
	}, func() {
		err := req.Err()
		t.setResult(op, keyvalue.OpResult{Op: op, Err: err})
	})
	if listenErr != nil {
		t.setResult(op, keyvalue.OpResult{Op: op, Err: listenErr})
		return
	}
	return
}

func (t *transaction) Set(name string, record keyvalue.FileRecord, contents blob.Blob) (op keyvalue.OpID) {
	op = t.newOp()
	_, err := t.set(op, name, record, contents)
	if err != nil {
		t.setResult(op, keyvalue.OpResult{Op: op, Err: err})
		return
	}
	return
}

func (t *transaction) set(op keyvalue.OpID, name string, record keyvalue.FileRecord, data blob.Blob) (*idb.Request, error) {
	infos, err := t.txn.ObjectStore(infoStore)
	if err != nil {
		return nil, err
	}
	contents, err := t.txn.ObjectStore(contentsStore)
	if err != nil {
		return nil, err
	}
	t.setResult(op, keyvalue.OpResult{Op: op}) // Ensure an op is recorded. A later result can overwrite it.

	if record == nil {
		if name == rootPath {
			return nil, hackpadfs.ErrNotImplemented // cannot delete root dir
		}
		req, err := deleteRecord(infos, contents, name)
		if err != nil {
			return nil, err
		}
		return req.Request, nil
	}

	if data != nil {
		// set file contents
		err := setFileContents(contents, name, data)
		if err != nil {
			return nil, err
		}
	}

	// always set metadata to update size when contents change
	req, parentExistsReq, err := validateAndSetFileMeta(t.ctx, infos, name, record, data)
	if err != nil {
		return nil, err
	}

	t.setPendingValidateErr(op, parentExistsReq)
	return req, nil
}

func (t *transaction) SetHandler(name string, record keyvalue.FileRecord, data blob.Blob, handler keyvalue.OpHandler) (op keyvalue.OpID) {
	op = t.newOp()
	req, err := t.set(op, name, record, data)
	if err != nil {
		t.setResult(op, keyvalue.OpResult{Op: op, Err: err})
		return
	}
	listenErr := req.Listen(t.ctx, func() {
		result := keyvalue.OpResult{Op: op, Err: req.Err()}
		if err := handler.Handle(t, result); err != nil {
			result.Err = err
		}
		t.setResult(op, result)
	}, func() {
		t.setResult(op, keyvalue.OpResult{Op: op, Err: req.Err()})
	})
	if listenErr != nil {
		t.setResult(op, keyvalue.OpResult{Op: op, Err: listenErr})
		return
	}
	return
}

func (t *transaction) Commit(ctx context.Context) ([]keyvalue.OpResult, error) {
	awaitErr := t.txn.Await(ctx)
	t.abort()
	for _, fn := range t.pendingResults {
		fn()
	}
	t.resultsMu.Lock()
	results := make([]keyvalue.OpResult, 0, len(t.results))
	for _, result := range t.results {
		results = append(results, result)
	}
	t.resultsMu.Unlock()
	sort.Slice(results, func(a, b int) bool {
		return results[a].Op < results[b].Op
	})
	return results, awaitErr
}

func (t *transaction) Abort() error {
	return t.txn.Abort()
}
