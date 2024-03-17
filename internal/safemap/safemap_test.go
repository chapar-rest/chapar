package safemap

import (
	"fmt"
	"sync"
	"testing"
)

func TestSafeMap(t *testing.T) {
	sm := New[int]()

	// Test Set and Get
	sm.Set("key1", 10)
	if val, ok := sm.Get("key1"); !ok || val != 10 {
		t.Errorf("Set or Get did not work as expected")
	}

	// Test Delete
	sm.Delete("key1")
	if _, ok := sm.Get("key1"); ok {
		t.Errorf("Delete did not work as expected")
	}

	// Test Len
	sm.Set("key2", 20)
	sm.Set("key3", 30)
	if mlen := sm.Len(); mlen != 2 {
		t.Errorf("Len did not work as expected, got %d, want %d", mlen, 2)
	}

	// Test Keys
	keys := sm.Keys()
	if len(keys) != 2 {
		t.Errorf("Keys did not work as expected, got %d, want %d", len(keys), 2)
	}

	// Test Has
	if !sm.Has("key2") {
		t.Errorf("Has did not work as expected")
	}
}

func TestSafeMap_Concurrency(t *testing.T) {
	sm := New[int]()

	// Test concurrency
	const numGoroutines = 100
	var wg sync.WaitGroup
	wg.Add(numGoroutines * 2) // For Set and Get operations
	for i := 0; i < numGoroutines; i++ {
		go func(i int) {
			defer wg.Done()
			key := fmt.Sprintf("key%d", i)
			sm.Set(key, i)
		}(i)
		go func(i int) {
			defer wg.Done()
			key := fmt.Sprintf("key%d", i)
			sm.Get(key) // We're not checking the return values here, just testing for race conditions
		}(i)
	}
	wg.Wait()

	if mlen := sm.Len(); mlen != numGoroutines {
		t.Errorf("Concurrent operations did not work as expected, got %d, want %d", mlen, numGoroutines)
	}
}
