package mem

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/hack-pad/hackpadfs"
	"github.com/hack-pad/hackpadfs/keyvalue"
	"github.com/hack-pad/hackpadfs/keyvalue/blob"
)

var _ keyvalue.TransactionStore = &store{}

type store struct {
	records sync.Map
	mu      sync.Mutex
}

func newStore() *store {
	return &store{}
}

type fileRecord struct {
	data    blob.Blob
	modTime time.Time
	store   *store
	path    string
	mode    hackpadfs.FileMode
}

func (f fileRecord) Data() (blob.Blob, error) {
	return f.data, nil
}

func (f fileRecord) Size() int64              { return int64(f.data.Len()) }
func (f fileRecord) Mode() hackpadfs.FileMode { return f.mode }
func (f fileRecord) ModTime() time.Time       { return f.modTime }
func (f fileRecord) Sys() interface{}         { return nil }

func (f fileRecord) ReadDirNames() ([]string, error) {
	if !f.mode.IsDir() {
		return nil, hackpadfs.ErrNotDir
	}
	var names []string
	prefix := f.path + "/"
	isRoot := f.path == "."
	if isRoot {
		prefix = ""
	}

	f.store.records.Range(func(key, _ interface{}) bool {
		p := key.(string)
		if strings.HasPrefix(p, prefix) {
			p = strings.TrimPrefix(p, prefix)
			if !strings.ContainsRune(p, '/') && !(isRoot && p == ".") {
				names = append(names, p)
			}
		}
		return true
	})
	return names, nil
}

func (s *store) Get(_ context.Context, path string) (keyvalue.FileRecord, error) {
	value, ok := s.records.Load(path)
	if !ok {
		return nil, hackpadfs.ErrNotExist
	}
	record := value.(keyvalue.FileRecord)
	return record, nil
}

func (s *store) Set(_ context.Context, path string, src keyvalue.FileRecord) error {
	var contents blob.Blob
	if src != nil {
		var err error
		contents, err = src.Data()
		if err != nil {
			return err
		}
	}
	return s.set(path, src, contents)
}

func (s *store) set(path string, src keyvalue.FileRecord, _ blob.Blob) error {
	if src == nil {
		s.records.Delete(path)
	} else {
		data, err := src.Data()
		if err != nil {
			return err
		}
		record := fileRecord{
			store:   s,
			path:    path,
			data:    data,
			mode:    src.Mode(),
			modTime: src.ModTime(),
		}
		s.records.Store(path, record)
	}
	return nil
}

type transaction struct {
	ctx     context.Context
	abort   context.CancelFunc
	store   *store
	results []keyvalue.OpResult
	op      keyvalue.OpID
}

func (s *store) Transaction(_ keyvalue.TransactionOptions) (keyvalue.Transaction, error) {
	ctx, cancel := context.WithCancel(context.Background())
	txn := &transaction{
		ctx:   ctx,
		abort: cancel,
		store: s,
	}
	s.mu.Lock()
	return txn, nil
}

func (t *transaction) prepOp() (keyvalue.OpID, error) {
	select {
	case <-t.ctx.Done():
		return 0, t.ctx.Err()
	default:
	}

	op := t.op
	t.op++
	return op, nil
}

func (t *transaction) Get(path string) keyvalue.OpID {
	return t.GetHandler(path, keyvalue.OpHandlerFunc(func(_ keyvalue.Transaction, _ keyvalue.OpResult) error {
		return nil
	}))
}

func (t *transaction) GetHandler(path string, handler keyvalue.OpHandler) keyvalue.OpID {
	op, err := t.prepOp()
	if err != nil {
		t.results = append(t.results, keyvalue.OpResult{Op: op, Err: err})
		return op
	}
	record, err := t.store.Get(t.ctx, path)
	result := keyvalue.OpResult{Op: op, Record: record, Err: err}
	err = handler.Handle(t, result)
	if result.Err == nil && err != nil {
		result.Err = err
	}
	t.results = append(t.results, result)
	return op
}

func (t *transaction) Set(path string, src keyvalue.FileRecord, contents blob.Blob) keyvalue.OpID {
	return t.SetHandler(path, src, contents, keyvalue.OpHandlerFunc(func(_ keyvalue.Transaction, _ keyvalue.OpResult) error {
		return nil
	}))
}

func (t *transaction) SetHandler(path string, src keyvalue.FileRecord, contents blob.Blob, handler keyvalue.OpHandler) keyvalue.OpID {
	op, err := t.prepOp()
	if err != nil {
		t.results = append(t.results, keyvalue.OpResult{Op: op, Err: err})
		return op
	}
	err = t.store.set(path, src, contents)
	result := keyvalue.OpResult{Op: op, Err: err}
	err = handler.Handle(t, result)
	if result.Err == nil && err != nil {
		result.Err = err
	}
	t.results = append(t.results, result)
	return op
}

func (t *transaction) Commit(_ context.Context) ([]keyvalue.OpResult, error) {
	t.abort()
	t.store.mu.Unlock()
	return t.results, nil
}

func (t *transaction) Abort() error {
	t.abort()
	t.store.mu.Unlock()
	return nil
}
