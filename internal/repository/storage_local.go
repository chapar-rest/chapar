package repository

import (
	"os"
	"path/filepath"
)

// LocalStorage is a StorageBackend that delegates to the local OS filesystem.
// It is the default backend used by NewFilesystemV2.
type LocalStorage struct{}

// NewLocalStorage returns a new LocalStorage.
func NewLocalStorage() *LocalStorage { return &LocalStorage{} }

func (l *LocalStorage) MkdirAll(path string) error {
	return os.MkdirAll(path, 0755)
}

func (l *LocalStorage) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func (l *LocalStorage) WriteFile(path string, data []byte) error {
	return os.WriteFile(path, data, 0644)
}

func (l *LocalStorage) Rename(oldPath, newPath string) error {
	return os.Rename(oldPath, newPath)
}

func (l *LocalStorage) Remove(path string) error {
	return os.Remove(path)
}

func (l *LocalStorage) RemoveAll(path string) error {
	return os.RemoveAll(path)
}

func (l *LocalStorage) Stat(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func (l *LocalStorage) ListEntries(path string) ([]StorageEntry, error) {
	des, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	entries := make([]StorageEntry, 0, len(des))
	for _, de := range des {
		entries = append(entries, StorageEntry{
			Name:  de.Name(),
			IsDir: de.IsDir(),
		})
	}
	return entries, nil
}

func (l *LocalStorage) Glob(pattern string) ([]string, error) {
	return filepath.Glob(pattern)
}
