package safemap

import "sync"

type Map[T any] struct {
	m   map[string]T
	mux sync.RWMutex
}

func New[T any]() *Map[T] {
	return &Map[T]{
		m: make(map[string]T),
	}
}

func (m *Map[T]) Set(key string, value T) {
	if m == nil {
		return
	}

	m.mux.Lock()
	m.m[key] = value
	m.mux.Unlock()
}

func (m *Map[T]) Get(key string) (T, bool) {
	if m == nil {
		var empty T
		return empty, false
	}

	m.mux.RLock()
	value, ok := m.m[key]
	m.mux.RUnlock()
	return value, ok
}

func (m *Map[T]) Keys() []string {
	if m == nil {
		return nil
	}

	m.mux.RLock()
	keys := make([]string, 0, len(m.m))
	for k := range m.m {
		keys = append(keys, k)
	}
	m.mux.RUnlock()
	return keys
}

func (m *Map[T]) Has(key string) bool {
	if m == nil {
		return false
	}

	m.mux.RLock()
	_, ok := m.m[key]
	m.mux.RUnlock()
	return ok
}

func (m *Map[T]) Delete(key string) {
	if m == nil {
		return
	}

	m.mux.Lock()
	delete(m.m, key)
	m.mux.Unlock()
}

func (m *Map[T]) Len() int {
	if m == nil {
		return 0
	}

	m.mux.RLock()
	length := len(m.m)
	m.mux.RUnlock()
	return length
}
