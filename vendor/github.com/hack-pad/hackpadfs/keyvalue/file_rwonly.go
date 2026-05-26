package keyvalue

import (
	"github.com/hack-pad/hackpadfs"
	"github.com/hack-pad/hackpadfs/keyvalue/blob"
)

type readOnlyFile struct {
	file *file
}

func (r *readOnlyFile) Close() error {
	return r.file.Close()
}

func (r *readOnlyFile) Read(p []byte) (n int, err error) {
	return r.file.Read(p)
}

func (r *readOnlyFile) ReadBlob(length int) (blob blob.Blob, n int, err error) {
	return r.file.ReadBlob(length)
}

func (r *readOnlyFile) ReadAt(p []byte, off int64) (n int, err error) {
	return r.file.ReadAt(p, off)
}

func (r *readOnlyFile) ReadBlobAt(length int, off int64) (b blob.Blob, n int, err error) {
	return r.file.ReadBlobAt(length, off)
}

func (r *readOnlyFile) Seek(offset int64, whence int) (int64, error) {
	return r.file.Seek(offset, whence)
}

func (r *readOnlyFile) Stat() (hackpadfs.FileInfo, error) {
	return r.file.Stat()
}

func (r *readOnlyFile) Truncate(size int64) error {
	return r.file.Truncate(size)
}

func (r *readOnlyFile) ReadDir(n int) ([]hackpadfs.DirEntry, error) {
	return r.file.ReadDir(n)
}

func (r *readOnlyFile) Chmod(mode hackpadfs.FileMode) error {
	return r.file.Chmod(mode)
}

type writeOnlyFile struct {
	file *file
}

func (w *writeOnlyFile) Read(_ []byte) (n int, err error) {
	// Read is required by hackpadfs.File
	return 0, &hackpadfs.PathError{Op: "read", Path: w.file.path, Err: hackpadfs.ErrNotImplemented}
}

func (w *writeOnlyFile) Close() error {
	return w.file.Close()
}

func (w *writeOnlyFile) Seek(offset int64, whence int) (int64, error) {
	return w.file.Seek(offset, whence)
}

func (w *writeOnlyFile) Write(p []byte) (n int, err error) {
	return w.file.Write(p)
}

func (w *writeOnlyFile) WriteBlob(p blob.Blob) (n int, err error) {
	return w.file.WriteBlob(p)
}

func (w *writeOnlyFile) WriteAt(p []byte, off int64) (n int, err error) {
	return w.file.WriteAt(p, off)
}

func (w *writeOnlyFile) WriteBlobAt(p blob.Blob, off int64) (n int, err error) {
	return w.file.WriteBlobAt(p, off)
}

func (w *writeOnlyFile) Stat() (hackpadfs.FileInfo, error) {
	return w.file.Stat()
}

func (w *writeOnlyFile) Truncate(size int64) error {
	return w.file.Truncate(size)
}

func (w *writeOnlyFile) Chmod(mode hackpadfs.FileMode) error {
	return w.file.Chmod(mode)
}
