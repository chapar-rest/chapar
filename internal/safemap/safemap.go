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
	m.mux.Lock()
	m.m[key] = value
	m.mux.Unlock()
}

func (m *Map[T]) Get(key string) (T, bool) {
	m.mux.RLock()
	value, ok := m.m[key]
	m.mux.RUnlock()
	return value, ok
}

func (m *Map[T]) Delete(key string) {
	m.mux.Lock()
	delete(m.m, key)
	m.mux.Unlock()
}

func (m *Map[T]) Len() int {
	m.mux.RLock()
	length := len(m.m)
	m.mux.RUnlock()
	return length
}
