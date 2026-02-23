package main

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// demoLockFreeCounter benchmarks an atomic counter against a mutex counter
// to show when atomics win and when the difference is negligible.
//
// Atomics shine for single-variable counters with no surrounding logic.
// A Mutex is better when you protect multiple variables or need a
// read-modify-write with complex logic.
func demoLockFreeCounter() {
	const goroutines = 8
	const increments = 100_000

	// ── Atomic ───────────────────────────────────────────────────────────────
	var atomicCount atomic.Int64
	var wg sync.WaitGroup

	start := time.Now()
	wg.Add(goroutines)
	for range goroutines {
		go func() {
			defer wg.Done()
			for range increments {
				atomicCount.Add(1)
			}
		}()
	}
	wg.Wait()
	atomicDur := time.Since(start)

	// ── Mutex ─────────────────────────────────────────────────────────────────
	var mu sync.Mutex
	var mutexCount int64

	start = time.Now()
	wg.Add(goroutines)
	for range goroutines {
		go func() {
			defer wg.Done()
			for range increments {
				mu.Lock()
				mutexCount++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	mutexDur := time.Since(start)

	expected := int64(goroutines * increments)
	fmt.Printf("  atomic: %d in %v\n", atomicCount.Load(), atomicDur.Round(time.Millisecond))
	fmt.Printf("  mutex:  %d in %v\n", mutexCount, mutexDur.Round(time.Millisecond))
	fmt.Printf("  expected: %d — both correct: %v\n",
		expected, atomicCount.Load() == expected && mutexCount == expected)
}

// demoShutdownFlag shows the canonical "shutdown flag" pattern:
// a single atomic.Bool signals all workers to stop cleanly.
//
// This avoids the overhead of closing a channel when the set of workers
// is not known in advance (e.g. spawned dynamically) and the signalling
// goroutine does not hold a reference to all of them.
func demoShutdownFlag() {
	var shutdown atomic.Bool
	var wg sync.WaitGroup

	for i := range 3 {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for tick := range 20 {
				if shutdown.Load() {
					fmt.Printf("  worker %d: saw shutdown at tick %d\n", id, tick)
					return
				}
				time.Sleep(10 * time.Millisecond)
			}
		}(i)
	}

	// Signal after 35 ms — workers typically see it at tick 3–4.
	time.Sleep(35 * time.Millisecond)
	shutdown.Store(true)
	fmt.Println("  shutdown flag set")

	wg.Wait()
	fmt.Println("  all workers stopped")
}

// SliceSnapshot is an immutable snapshot of a slice used for copy-on-write.
type SliceSnapshot struct {
	Items []string
}

// demoCopyOnWrite shows the copy-on-write (COW) pattern with atomic.Pointer:
// readers always get a consistent snapshot; writers replace the whole slice
// atomically rather than mutating it under a lock.
//
// Trade-off: writes are more expensive (clone + replace) but reads are
// lock-free and never block writers.
func demoCopyOnWrite() {
	var snap atomic.Pointer[SliceSnapshot]
	snap.Store(&SliceSnapshot{Items: []string{"a", "b", "c"}})

	// append atomically: load, clone, append, CAS-replace.
	appendItem := func(item string) {
		for {
			old := snap.Load()
			// Build a new slice — old is never modified.
			newItems := make([]string, len(old.Items)+1)
			copy(newItems, old.Items)
			newItems[len(old.Items)] = item
			next := &SliceSnapshot{Items: newItems}
			if snap.CompareAndSwap(old, next) {
				return // won the race
			}
			// Lost to another writer — retry with the latest snapshot.
		}
	}

	var wg sync.WaitGroup
	for _, item := range []string{"d", "e", "f"} {
		wg.Add(1)
		go func(v string) {
			defer wg.Done()
			appendItem(v)
		}(item)
	}
	wg.Wait()

	current := snap.Load()
	fmt.Printf("  snapshot after concurrent appends (%d items): %v\n",
		len(current.Items), current.Items)
}
