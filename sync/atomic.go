package main

import (
	"fmt"
	"sync"
	"sync/atomic"
)

// demoAtomic shows sync/atomic: lock-free operations on primitive types.
//
// Atomic operations are cheaper than a mutex for simple counters and flags
// because they map to single CPU instructions (no OS involvement).
//
// Go 1.19+ provides typed wrappers (atomic.Int64, atomic.Bool, etc.) that
// are safer and more ergonomic than the older function-based API.
func demoAtomic() {
	demoAtomicCounter()
	demoAtomicCAS()
}

// demoAtomicCounter shows atomic increment with the typed atomic.Int64 API.
func demoAtomicCounter() {
	var counter atomic.Int64
	var wg sync.WaitGroup

	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			counter.Add(1) // atomic: no race, no mutex needed
		}()
	}
	wg.Wait()

	fmt.Println("  counter:", counter.Load()) // always 1000

	// Other operations on typed atomics:
	counter.Store(0)              // unconditional write
	prev := counter.Swap(99)     // write and return old value
	fmt.Printf("  after Swap(99): prev=%d current=%d\n", prev, counter.Load())

	// atomic.Bool
	var flag atomic.Bool
	flag.Store(true)
	fmt.Println("  flag:", flag.Load())
}

// demoAtomicCAS shows Compare-And-Swap (CAS): atomically sets a value only if
// it currently equals the expected value. The fundamental primitive behind
// lock-free data structures and spin-locks.
func demoAtomicCAS() {
	var state atomic.Int32 // 0=idle, 1=running, 2=done

	// Transition idle → running: succeeds because state is 0.
	swapped := state.CompareAndSwap(0, 1)
	fmt.Printf("  CAS(0→1): swapped=%v state=%d\n", swapped, state.Load())

	// Transition idle → running again: fails because state is now 1, not 0.
	swapped = state.CompareAndSwap(0, 1)
	fmt.Printf("  CAS(0→1): swapped=%v state=%d\n", swapped, state.Load())

	// Transition running → done: succeeds.
	swapped = state.CompareAndSwap(1, 2)
	fmt.Printf("  CAS(1→2): swapped=%v state=%d\n", swapped, state.Load())
}

// demoAtomicValue shows atomic.Value: store and load an arbitrary immutable
// value atomically. Useful for config hot-reload, routing tables, snapshots.
//
// Rules:
//   - All values stored must be the same concrete type.
//   - The stored value should be treated as immutable; never modify it after storing.
func demoAtomicValue() {
	type Config struct {
		MaxConns int
		Timeout  int
	}

	var cfg atomic.Value

	// Initial config.
	cfg.Store(Config{MaxConns: 10, Timeout: 30})

	// Any goroutine can read the config without locking.
	c := cfg.Load().(Config)
	fmt.Printf("  config v1: maxConns=%d timeout=%d\n", c.MaxConns, c.Timeout)

	// Hot-reload: replace the entire config atomically.
	// Readers either see the old or the new value — never a partial update.
	cfg.Store(Config{MaxConns: 50, Timeout: 5})

	c = cfg.Load().(Config)
	fmt.Printf("  config v2: maxConns=%d timeout=%d\n", c.MaxConns, c.Timeout)

	// Swap: store and return the previous value in one atomic operation.
	prev := cfg.Swap(Config{MaxConns: 100, Timeout: 1}).(Config)
	fmt.Printf("  after Swap: prev maxConns=%d  new maxConns=%d\n",
		prev.MaxConns, cfg.Load().(Config).MaxConns)

	// CompareAndSwap: replace only if current value equals the expected one.
	current := cfg.Load().(Config)
	swapped := cfg.CompareAndSwap(current, Config{MaxConns: 200, Timeout: 1})
	fmt.Printf("  CAS: swapped=%v  maxConns=%d\n", swapped, cfg.Load().(Config).MaxConns)
}
