package repository

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// MemStorage is an in-memory StorageBackend intended for use in tests only.
// It is never compiled into production binaries.
type MemStorage struct {
	mu   sync.RWMutex
	data map[string][]byte // full path → file content
	dirs map[string]bool   // full path → true (is a directory)
}

// NewMemStorage returns an empty MemStorage.
func NewMemStorage() *MemStorage {
	return &MemStorage{
		data: make(map[string][]byte),
		dirs: make(map[string]bool),
	}
}

func (m *MemStorage) MkdirAll(path string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	// Mark path and every ancestor as a directory.
	for p := filepath.Clean(path); p != filepath.Dir(p); p = filepath.Dir(p) {
		m.dirs[p] = true
	}
	m.dirs[filepath.Clean(path)] = true
	return nil
}

func (m *MemStorage) ReadFile(path string) ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	data, ok := m.data[path]
	if !ok {
		return nil, &os.PathError{Op: "open", Path: path, Err: os.ErrNotExist}
	}
	result := make([]byte, len(data))
	copy(result, data)
	return result, nil
}

func (m *MemStorage) WriteFile(path string, data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	content := make([]byte, len(data))
	copy(content, data)
	m.data[path] = content
	// Auto-create parent directories.
	for p := filepath.Dir(path); p != filepath.Dir(p); p = filepath.Dir(p) {
		m.dirs[p] = true
	}
	return nil
}

func (m *MemStorage) Rename(oldPath, newPath string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// File rename.
	if data, ok := m.data[oldPath]; ok {
		m.data[newPath] = data
		delete(m.data, oldPath)
		return nil
	}

	// Directory rename: move all files and subdirs under oldPath.
	if m.dirs[oldPath] {
		prefix := oldPath + string(os.PathSeparator)
		for p, d := range m.data {
			if strings.HasPrefix(p, prefix) {
				newP := newPath + p[len(oldPath):]
				m.data[newP] = d
				delete(m.data, p)
			}
		}
		for p := range m.dirs {
			if p == oldPath || strings.HasPrefix(p, prefix) {
				newP := newPath + p[len(oldPath):]
				m.dirs[newP] = true
				delete(m.dirs, p)
			}
		}
		return nil
	}

	return fmt.Errorf("rename %s %s: no such file or directory", oldPath, newPath)
}

func (m *MemStorage) Remove(path string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.data[path]; !ok {
		return &os.PathError{Op: "remove", Path: path, Err: os.ErrNotExist}
	}
	delete(m.data, path)
	return nil
}

func (m *MemStorage) RemoveAll(path string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	prefix := path + string(os.PathSeparator)
	for p := range m.data {
		if p == path || strings.HasPrefix(p, prefix) {
			delete(m.data, p)
		}
	}
	for p := range m.dirs {
		if p == path || strings.HasPrefix(p, prefix) {
			delete(m.dirs, p)
		}
	}
	return nil
}

func (m *MemStorage) Stat(path string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if _, ok := m.data[path]; ok {
		return true, nil
	}
	return m.dirs[path], nil
}

func (m *MemStorage) ListEntries(path string) ([]StorageEntry, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	path = filepath.Clean(path)
	if !m.dirs[path] {
		return nil, fmt.Errorf("listentries %s: directory does not exist", path)
	}

	seen := make(map[string]bool)
	var entries []StorageEntry

	for p := range m.data {
		if filepath.Clean(filepath.Dir(p)) == path {
			name := filepath.Base(p)
			if !seen[name] {
				seen[name] = true
				entries = append(entries, StorageEntry{Name: name, IsDir: false})
			}
		}
	}
	for p := range m.dirs {
		if p == path {
			continue
		}
		if filepath.Clean(filepath.Dir(p)) == path {
			name := filepath.Base(p)
			if !seen[name] {
				seen[name] = true
				entries = append(entries, StorageEntry{Name: name, IsDir: true})
			}
		}
	}

	return entries, nil
}

func (m *MemStorage) Glob(pattern string) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var matches []string
	for p := range m.data {
		matched, err := filepath.Match(pattern, p)
		if err != nil {
			return nil, err
		}
		if matched {
			matches = append(matches, p)
		}
	}
	sort.Strings(matches)
	return matches, nil
}
