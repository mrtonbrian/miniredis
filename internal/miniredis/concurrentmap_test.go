//go:build test
// +build test

package miniredis

import (
	"sync"
	"testing"
)

func TestConcurrentMap_BasicOperations(t *testing.T) {
	cm := NewConcurrentMap[string, int]()

	key := "test"
	value := 42
	cm.Set(&key, &value)

	if got, exists := cm.Get(&key); !exists || got != value {
		t.Errorf("Get() = %v, %v; want %v, true", got, exists, value)
	}

	nonExistentKey := "missing"
	if got, exists := cm.Get(&nonExistentKey); exists || got != 0 {
		t.Errorf("Get() for non-existent key = %v, %v; want 0, false", got, exists)
	}

	cm.Delete(&key)
	if _, exists := cm.Get(&key); exists {
		t.Error("Delete() failed, key still exists")
	}
}

func TestConcurrentMap_Update(t *testing.T) {
	cm := NewConcurrentMap[string, int]()
	key := "counter"
	value := 1
	cm.Set(&key, &value)

	success := cm.Update(&key, func(v int) int {
		return v + 1
	})

	if !success {
		t.Error("Update() returned false for existing key")
	}

	if got, _ := cm.Get(&key); got != 2 {
		t.Errorf("Update() failed to modify value: got %v, want 2", got)
	}

	missingKey := "missing"
	success = cm.Update(&missingKey, func(v int) int {
		return v + 1
	})

	if success {
		t.Error("Update() returned true for non-existent key")
	}
}

func TestConcurrentMap_Concurrency(t *testing.T) {
	cm := NewConcurrentMap[string, int]()
	key := "counter"
	initial := 0
	cm.Set(&key, &initial)

	const numGoroutines = 100
	const incrementsPerGoroutine = 100
	var wg sync.WaitGroup

	// Launch multiple goroutines to increment the counter
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < incrementsPerGoroutine; j++ {
				cm.Update(&key, func(v int) int {
					return v + 1
				})
			}
		}()
	}

	wg.Wait()

	// Check final value
	if final, _ := cm.Get(&key); final != numGoroutines*incrementsPerGoroutine {
		t.Errorf("Concurrent updates failed: got %v, want %v",
			final, numGoroutines*incrementsPerGoroutine)
	}
}
