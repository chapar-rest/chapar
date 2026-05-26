package keyvalue

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/hack-pad/hackpadfs/keyvalue/blob"
)

// TransactionStore is a Store that can create a Transaction.
type TransactionStore interface {
	Store
	Transaction(options TransactionOptions) (Transaction, error)
}

// TransactionMode is the kind of transaction mode, i.e. read-only or read-write
type TransactionMode int

// Transaction modes
const (
	TransactionReadOnly TransactionMode = iota
	TransactionReadWrite
)

// TransactionOptions contain options used to construct a Transaction from a Store
type TransactionOptions struct {
	Mode TransactionMode
}

// OpID is a unique ID within the transaction that generated it. It's used to correlate which Get/Set operation produced which result.
type OpID int64

// OpResult is returned from Transaction.Commit(), representing an operation's result with any data or error it produced.
type OpResult struct {
	Record FileRecord
	Err    error
	Op     OpID
}

// OpHandler processes 'result' during the commit process of 'txn'.
// If the transaction should not proceed, the handler should call txn.Abort().
type OpHandler interface {
	Handle(txn Transaction, result OpResult) error
}

// OpHandlerFunc is a convenient func wrapper for implementing OpHandler
type OpHandlerFunc func(txn Transaction, result OpResult) error

// Handle implements OpHandler
func (o OpHandlerFunc) Handle(txn Transaction, result OpResult) error {
	return o(txn, result)
}

// Transaction behaves like a Store but only returns results after running Commit().
// GetHandler and SetHandler can be used to interrupt transaction processing and handle the response,
// permitting an opportunity to Abort() or perform more operations.
type Transaction interface {
	Get(path string) OpID
	GetHandler(path string, handler OpHandler) OpID
	Set(path string, src FileRecord, contents blob.Blob) OpID
	SetHandler(path string, src FileRecord, contents blob.Blob, handler OpHandler) OpID
	Commit(ctx context.Context) ([]OpResult, error)
	Abort() error
}

type unsafeSerialTransaction struct {
	ctx       context.Context
	abort     context.CancelFunc
	store     Store
	results   map[OpID]OpResult
	resultsMu sync.Mutex
	nextOp    OpID
}

// TransactionOrSerial attempts to produce a Transaction from 'store'.
// If unsupported, returns an unsafe transaction instead, which runs each action serially without transactional safety.
//
// This is used in FS to attempt transactions whenever possible.
// Since some Stores don't need transactions, they aren't required to implement TransactionStore.
func TransactionOrSerial(store Store, options TransactionOptions) (Transaction, error) {
	if store, ok := store.(TransactionStore); ok {
		return store.Transaction(options)
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &unsafeSerialTransaction{
		ctx:     ctx,
		abort:   cancel,
		store:   store,
		results: make(map[OpID]OpResult),
	}, nil
}

func (u *unsafeSerialTransaction) newOp() OpID {
	nextOp := atomic.AddInt64((*int64)(&u.nextOp), 1)
	return OpID(nextOp - 1)
}

func (u *unsafeSerialTransaction) setResult(op OpID, result OpResult) {
	u.resultsMu.Lock()
	u.results[op] = result
	u.resultsMu.Unlock()
}

func abortErr(ctx, extraCtx context.Context) error {
	if extraCtx == nil {
		extraCtx = context.Background()
	}
	select {
	case <-extraCtx.Done():
		return extraCtx.Err()
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}

func (u *unsafeSerialTransaction) Get(path string) OpID {
	return u.GetHandler(path, OpHandlerFunc(func(_ Transaction, _ OpResult) error {
		return nil
	}))
}

func (u *unsafeSerialTransaction) GetHandler(path string, handler OpHandler) OpID {
	op := u.newOp()
	if err := abortErr(u.ctx, nil); err != nil {
		u.setResult(op, OpResult{Op: op, Err: err})
		return op
	}

	record, err := u.store.Get(u.ctx, path)
	result := OpResult{Op: op, Record: record, Err: err}
	err = handler.Handle(u, result)
	if result.Err == nil && err != nil {
		result.Err = err
	}
	u.setResult(op, result)
	return op
}

func (u *unsafeSerialTransaction) Set(path string, src FileRecord, contents blob.Blob) OpID {
	return u.SetHandler(path, src, contents, OpHandlerFunc(func(_ Transaction, _ OpResult) error {
		return nil
	}))
}

func (u *unsafeSerialTransaction) SetHandler(path string, src FileRecord, _ blob.Blob, handler OpHandler) OpID {
	op := u.newOp()
	if err := abortErr(u.ctx, nil); err != nil {
		u.setResult(op, OpResult{Op: op, Err: err})
		return op
	}

	err := u.store.Set(u.ctx, path, src)
	result := OpResult{Op: op, Err: err}
	err = handler.Handle(u, result)
	if result.Err == nil && err != nil {
		result.Err = err
	}
	u.setResult(op, result)
	return op
}

func (u *unsafeSerialTransaction) Commit(ctx context.Context) ([]OpResult, error) {
	if err := abortErr(u.ctx, ctx); err != nil {
		return nil, err
	}
	u.abort()
	opCount := atomic.LoadInt64((*int64)(&u.nextOp))
	results := make([]OpResult, opCount)
	u.resultsMu.Lock()
	for op, result := range u.results {
		results[op] = result
	}
	u.resultsMu.Unlock()
	return results, nil
}

func (u *unsafeSerialTransaction) Abort() error {
	u.abort()
	return nil
}
