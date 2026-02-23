package main

import (
	"fmt"
	"sync"
)

// demoSyncMap shows sync.Map: a concurrent-safe map with no external locking.
//
// Prefer sync.Map over map+RWMutex when:
//   - Keys are written once and read many times (e.g. caches).
//   - Multiple goroutines write to disjoint sets of keys.
//
// For general-purpose concurrent maps with heavy mixed reads/writes,
// a sharded map+Mutex often performs better.
func demoSyncMap() {
	var m sync.Map

	var wg sync.WaitGroup

	// Store: concurrent writes to different keys.
	services := []struct{ name, addr string }{
		{"payments", "10.0.0.1:8080"},
		{"shipping", "10.0.0.2:8080"},
		{"inventory", "10.0.0.3:8080"},
	}
	for _, svc := range services {
		wg.Add(1)
		go func(name, addr string) {
			defer wg.Done()
			m.Store(name, addr)
		}(svc.name, svc.addr)
	}
	wg.Wait()

	// Load: retrieve a value; ok=false if key absent.
	if addr, ok := m.Load("payments"); ok {
		fmt.Println("  payments:", addr)
	}
	if _, ok := m.Load("unknown"); !ok {
		fmt.Println("  unknown: not found")
	}

	// LoadOrStore: atomic get-or-set. Returns existing value if present.
	actual, loaded := m.LoadOrStore("payments", "NEW_ADDR")
	fmt.Printf("  LoadOrStore payments: value=%v  loaded=%v\n", actual, loaded)

	actual, loaded = m.LoadOrStore("reviews", "10.0.0.4:8080")
	fmt.Printf("  LoadOrStore reviews:  value=%v  loaded=%v\n", actual, loaded)

	// Delete: remove a key.
	m.Delete("inventory")

	// Range: iterate over all key-value pairs (order is not guaranteed).
	// Return false from the callback to stop early.
	fmt.Println("  range:")
	m.Range(func(key, value any) bool {
		fmt.Printf("    %s → %s\n", key, value)
		return true // continue iteration
	})

	// LoadAndDelete: atomic load + delete in one operation.
	if val, ok := m.LoadAndDelete("shipping"); ok {
		fmt.Println("  LoadAndDelete shipping:", val)
	}
}
