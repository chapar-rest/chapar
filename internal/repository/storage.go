package repository

import "errors"

// ErrNotImplemented is returned by stub backends that have not yet been implemented.
var ErrNotImplemented = errors.New("storage backend: not implemented")

// StorageEntry represents a single entry (file or directory) returned by ListEntries.
type StorageEntry struct {
	Name  string
	IsDir bool
}

// StorageBackend abstracts file I/O so that FilesystemV2 does not depend on the
// local OS. All path arguments use the OS-native separator as constructed by
// filepath.Join — individual backends are responsible for any further normalization
// they require (e.g. converting to S3 key format).
//
// Implementations must be safe for concurrent use.
type StorageBackend interface {
	// MkdirAll creates the directory named by path, along with any necessary
	// parents. It is not an error if the directory already exists.
	MkdirAll(path string) error

	// ReadFile reads the named file and returns its contents.
	ReadFile(path string) ([]byte, error)

	// WriteFile writes data to the named file, creating it if necessary.
	// If the file exists it is truncated before writing.
	WriteFile(path string, data []byte) error

	// Rename renames (moves) oldPath to newPath.
	Rename(oldPath, newPath string) error

	// Remove removes the named file. It is an error if path is a directory.
	Remove(path string) error

	// RemoveAll removes path and any children it contains, similar to os.RemoveAll.
	RemoveAll(path string) error

	// Stat reports whether the path exists.
	// Returns (true, nil) if exists, (false, nil) if not found, (false, err) on I/O errors.
	Stat(path string) (exists bool, err error)

	// ListEntries returns the direct children (files and directories) of path.
	ListEntries(path string) ([]StorageEntry, error)

	// Glob returns the names of files matching pattern using filepath.Match syntax.
	Glob(pattern string) ([]string, error)
}
