package main

import (
	"fmt"
	"sync"
	"sync/atomic"
)

// demoInt64 shows the typed atomic.Int64 API (Go 1.19+).
//
// All methods are guaranteed to be atomic — no torn reads, no lost writes,
// no reordering across the operation boundary (sequential consistency).
//
// Use the typed API (atomic.Int64, atomic.Bool, …) rather than the older
// function API (atomic.AddInt64, atomic.LoadInt64, …): it's safer because
// you cannot accidentally pass a non-aligned pointer, and the zero value is
// ready to use.
func demoInt64() {
	var n atomic.Int64 // zero value is 0, no init needed

	// Add returns the NEW value after the addition.
	fmt.Println("  Add(10):", n.Add(10))  // 10
	fmt.Println("  Add(5): ", n.Add(5))   // 15
	fmt.Println("  Add(-3):", n.Add(-3))  // 12
	fmt.Println("  Load():  ", n.Load())  // 12

	n.Store(100)
	fmt.Println("  Store(100), Load():", n.Load()) // 100

	old := n.Swap(42)
	fmt.Printf("  Swap(42): old=%d new=%d\n", old, n.Load()) // old=100, new=42

	// Concurrent usage: 100 goroutines each add 1 → expect exactly 100.
	var counter atomic.Int64
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			counter.Add(1)
		}()
	}
	wg.Wait()
	fmt.Printf("  concurrent Add ×100 → %d (expected 100)\n", counter.Load())
}

// demoUint64 shows atomic.Uint64 — same API as Int64 but unsigned.
//
// Common use: monotonic event counters, sequence numbers, byte counters.
// Overflow wraps around (same as uint64 arithmetic).
func demoUint64() {
	var seq atomic.Uint64

	for i := range 5 {
		id := seq.Add(1) // pre-increment: returns new value
		fmt.Printf("  sequence id %d (i=%d)\n", id, i)
	}

	// Wrap-around: max uint64 + 1 == 0.
	var wrap atomic.Uint64
	wrap.Store(^uint64(0)) // max uint64
	wrap.Add(1)
	fmt.Printf("  max uint64 + 1 = %d (wrapped)\n", wrap.Load())
}

// demoBool shows atomic.Bool — the most common flag type.
//
// Typical uses: "is the server shutting down?", "has the task started?".
func demoBool() {
	var started atomic.Bool

	fmt.Println("  started:", started.Load()) // false

	started.Store(true)
	fmt.Println("  started:", started.Load()) // true

	// Swap returns the old value.
	old := started.Swap(false)
	fmt.Printf("  Swap(false): old=%v, new=%v\n", old, started.Load())
}

// demoCAS shows Compare-And-Swap (CAS): the building block of lock-free
// algorithms.
//
// CAS(ptr, old, new) atomically:
//  1. Reads the current value at ptr.
//  2. If it equals old, writes new and returns true.
//  3. Otherwise, leaves ptr unchanged and returns false.
//
// A CAS loop retries until it wins the race — this is an optimistic spin.
func demoCAS() {
	var val atomic.Int64
	val.Store(10)

	// Simple CAS: succeeds because current value == 10.
	swapped := val.CompareAndSwap(10, 20)
	fmt.Printf("  CAS(10→20): swapped=%v, val=%d\n", swapped, val.Load())

	// Fails: current value is now 20, not 10.
	swapped = val.CompareAndSwap(10, 99)
	fmt.Printf("  CAS(10→99): swapped=%v, val=%d\n", swapped, val.Load())

	// CAS loop: safely increment without Add (shows the pattern).
	// Useful when you need read-modify-write with arbitrary logic.
	var counter atomic.Int64
	counter.Store(0)

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				old := counter.Load()
				// Only proceed if we win the CAS; otherwise retry.
				if counter.CompareAndSwap(old, old+1) {
					return
				}
			}
		}()
	}
	wg.Wait()
	fmt.Printf("  CAS-loop ×50 → %d (expected 50)\n", counter.Load())
}
