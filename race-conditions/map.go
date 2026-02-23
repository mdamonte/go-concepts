package main

import (
	"fmt"
	"sync"
)

// demoMapRace explains why concurrent map access is especially dangerous in Go.
//
// Unlike a counter race (which silently loses updates), a concurrent map
// write causes a FATAL runtime error that cannot be recovered:
//
//	fatal error: concurrent map read and map write
//
// The Go runtime detects this and kills the program immediately.
// This is intentional: a corrupted map's internal structure can cause
// arbitrary memory corruption, so Go fails fast.
//
// ── What the racy code looks like (DO NOT run without synchronization) ──────
//
//	m := make(map[string]int)
//	go func() { m["a"]++ }()  // writer
//	go func() { _ = m["a"] }() // reader — fatal if concurrent with writer
//
// ── Detect it with the race detector ────────────────────────────────────────
//
//	go run -race .
//
// The race detector will flag the access before the fatal error even occurs.
func demoMapRace() {
	fmt.Println("  racy map code shown below — not executed to avoid fatal crash:")
	fmt.Println(`
  m := make(map[string]int)
  go func() { m["a"]++ }()   // writer goroutine
  go func() { _ = m["b"] }() // reader goroutine
  // → fatal error: concurrent map read and map write`)
}

// demoMapRWMutex fixes concurrent map access with sync.RWMutex.
// Multiple goroutines may hold a read lock simultaneously;
// a writer gets exclusive access.
//
// Use this when you need full control over the map type or key/value types.
func demoMapRWMutex() {
	var mu sync.RWMutex
	m := make(map[string]int)
	var wg sync.WaitGroup

	// 5 concurrent writers.
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			key := fmt.Sprintf("key%d", id)
			mu.Lock()
			m[key] = id * 10 // exclusive write
			mu.Unlock()
		}(i)
	}

	// 5 concurrent readers.
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			key := fmt.Sprintf("key%d", id)
			mu.RLock()
			_ = m[key] // shared read — safe alongside other RLocks
			mu.RUnlock()
		}(i)
	}

	wg.Wait()

	mu.RLock()
	fmt.Printf("  map has %d entries  ✓\n", len(m))
	mu.RUnlock()
}

// demoMapSyncMap fixes concurrent map access with sync.Map.
// No external locking required; the map handles synchronization internally.
//
// Best for: write-once-read-many caches, disjoint key sets per goroutine.
// Avoid for: high write churn on the same keys (sync.Map has more overhead
// than a plain map+RWMutex in write-heavy workloads).
func demoMapSyncMap() {
	var m sync.Map
	var wg sync.WaitGroup

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			m.Store(fmt.Sprintf("key%d", id), id*10) // no external lock needed
		}(i)
	}
	wg.Wait()

	count := 0
	m.Range(func(_, _ any) bool { count++; return true })
	fmt.Printf("  sync.Map has %d entries  ✓\n", count)
}
