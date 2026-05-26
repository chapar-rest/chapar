//go:build wasm
// +build wasm

// Package indexeddb contains a WebAssembly compatible file system. Uses IndexedDB under the hood.
package indexeddb

import (
	"context"
	"time"

	"github.com/hack-pad/go-indexeddb/idb"
	"github.com/hack-pad/hackpadfs"
	"github.com/hack-pad/hackpadfs/keyvalue"
	"github.com/hack-pad/safejs"
)

const (
	fsVersion = 1

	contentsStore = "contents"
	infoStore     = "info"
	parentKey     = "Parent"
)

// FS is a browser-based file system, storing files and metadata inside IndexedDB.
type FS struct {
	kv *keyvalue.FS
	db *idb.Database
}

// Options provides configuration options for a new FS.
type Options struct {
	Factory               *idb.Factory
	TransactionDurability idb.TransactionDurability
}

// NewFS returns a new FS.
func NewFS(ctx context.Context, name string, options Options) (*FS, error) {
	if options.Factory == nil {
		options.Factory = idb.Global()
	}
	openRequest, err := options.Factory.Open(ctx, name, fsVersion, func(db *idb.Database, _, _ uint) error {
		_, err := db.CreateObjectStore(contentsStore, idb.ObjectStoreOptions{})
		if err != nil {
			return err
		}
		infos, err := db.CreateObjectStore(infoStore, idb.ObjectStoreOptions{})
		if err != nil {
			return err
		}
		jsParentKey, err := safejs.ValueOf(parentKey)
		if err != nil {
			return err
		}
		_, err = infos.CreateIndex(parentKey, safejs.Unsafe(jsParentKey), idb.IndexOptions{})
		return err
	})
	if err != nil {
		return nil, err
	}
	db, err := openRequest.Await(ctx)
	if err != nil {
		return nil, err
	}
	kv, err := keyvalue.NewFS(newStore(db, options))
	return &FS{
		kv: kv,
		db: db,
	}, err
}

// Clear dangerously destroys all data inside this FS. Use with caution.
func (fs *FS) Clear(ctx context.Context) error {
	stores := []string{contentsStore, infoStore}
	txn, err := fs.db.Transaction(idb.TransactionReadWrite, stores[0], stores[1:]...)
	if err != nil {
		return err
	}
	for _, name := range stores {
		store, err := txn.ObjectStore(name)
		if err != nil {
			return err
		}
		_, err = store.Clear()
		if err != nil {
			return err
		}
	}
	err = txn.Commit()
	if err != nil {
		return err
	}
	err = txn.Await(ctx)
	if err != nil {
		return err
	}
	return fs.Mkdir(".", 0o666)
}

// Open implements hackpadfs.FS
func (fs *FS) Open(name string) (hackpadfs.File, error) {
	return fs.kv.Open(name)
}

// OpenFile implements hackpadfs.OpenFileFS
func (fs *FS) OpenFile(name string, flag int, perm hackpadfs.FileMode) (hackpadfs.File, error) {
	return fs.kv.OpenFile(name, flag, perm)
}

// Mkdir implements hackpadfs.MkdirFS
func (fs *FS) Mkdir(name string, perm hackpadfs.FileMode) error {
	return fs.kv.Mkdir(name, perm)
}

// MkdirAll implements hackpadfs.MkdirAllFS
func (fs *FS) MkdirAll(path string, perm hackpadfs.FileMode) error {
	return fs.kv.MkdirAll(path, perm)
}

// Remove implements hackpadfs.RemoveFS
func (fs *FS) Remove(name string) error {
	return fs.kv.Remove(name)
}

// Rename implements hackpadfs.RenameFS
func (fs *FS) Rename(oldname, newname string) error {
	return fs.kv.Rename(oldname, newname)
}

// Stat implements hackpadfs.StatFS
func (fs *FS) Stat(name string) (hackpadfs.FileInfo, error) {
	return fs.kv.Stat(name)
}

// Chmod implements hackpadfs.ChmodFS
func (fs *FS) Chmod(name string, mode hackpadfs.FileMode) error {
	return fs.kv.Chmod(name, mode)
}

// Chtimes implements hackpadfs.ChtimesFS
func (fs *FS) Chtimes(name string, atime time.Time, mtime time.Time) error {
	return fs.kv.Chtimes(name, atime, mtime)
}
