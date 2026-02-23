package main

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"
)

// Publication hazard: a goroutine publishes a value (sets a pointer or flag)
// before all the writes to the value's fields are visible to other goroutines.
//
// Modern CPUs and compilers reorder memory operations for performance.
// Without explicit synchronization, another goroutine can observe a pointer
// that is non-nil while the struct it points to is only partially initialized.
//
// This is NOT caught by `go run -race` because both accesses may be in
// separate goroutines with no interleaving — the issue is memory ordering,
// not a data race in the traditional sense.

type config struct {
	host    string
	port    int
	timeout time.Duration
}

// ── Racy singleton ────────────────────────────────────────────────────────────
//
// Double-checked locking WITHOUT proper synchronization.
// The race: a goroutine may see instance != nil while the struct fields
// written by the initializing goroutine are not yet visible (CPU store
// buffer hasn't flushed, or the compiler reordered the stores).

var (
	instance *config // shared pointer — racy without sync
	initMu   sync.Mutex
)

// getConfigRacy illustrates the anti-pattern.
// The outer check (`instance != nil`) is an unsynchronized read.
// Even if instance is non-nil, its fields may be zero.
func getConfigRacy() *config {
	if instance != nil { // unsynchronized read — DATA RACE
		return instance
	}
	initMu.Lock()
	defer initMu.Unlock()
	if instance == nil {
		instance = &config{
			host:    "localhost",
			port:    5432,
			timeout: 30 * time.Second,
		}
		// The pointer store and the field stores can be reordered by the CPU.
		// Another goroutine calling getConfigRacy may observe instance != nil
		// while host/port/timeout still contain their zero values.
	}
	return instance
}

// demoPublishRace shows that getConfigRacy can return a non-nil pointer
// whose fields are not yet visible. Detected by -race.
func demoPublishRace() {
	instance = nil // reset for demo
	fmt.Println("  racy double-checked locking — shown but not safe to run concurrently:")
	fmt.Println(`
  if instance != nil {     // unsynchronized read — DATA RACE
      return instance       // may return partially initialized struct
  }
  mu.Lock()
  if instance == nil {
      instance = &config{...} // stores may be reordered
  }
  mu.Unlock()`)

	// Safe sequential call to show the racy function returns the right value
	// when called from a single goroutine.
	cfg := getConfigRacy()
	fmt.Printf("  (sequential call) host=%s port=%d\n", cfg.host, cfg.port)
}

// ── Fix 1: sync.Once ─────────────────────────────────────────────────────────
//
// sync.Once guarantees:
//  1. The function runs exactly once.
//  2. All goroutines that observe the Once as "done" will see all writes
//     made by the initializing goroutine (happens-before guarantee).

var (
	cfgOnce     sync.Once
	cfgInstance *config
)

func getConfigOnce() *config {
	cfgOnce.Do(func() {
		cfgInstance = &config{
			host:    "localhost",
			port:    5432,
			timeout: 30 * time.Second,
		}
	})
	return cfgInstance
}

// demoPublishFixed shows that Once serializes initialization and provides
// the memory ordering guarantees needed to safely publish the pointer.
func demoPublishFixed() {
	var wg sync.WaitGroup
	results := make([]*config, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			results[id] = getConfigOnce() // safe: Once provides happens-before
		}(i)
	}
	wg.Wait()

	// All goroutines must receive the same pointer with fully visible fields.
	first := results[0]
	allSame := true
	for _, r := range results {
		if r != first || r.host == "" || r.port == 0 {
			allSame = false
		}
	}
	fmt.Printf("  all goroutines got same config: %v  host=%s port=%d  ✓\n",
		allSame, first.host, first.port)
}

// ── Fix 2: atomic.Pointer (Go 1.19+) ─────────────────────────────────────────
//
// For hot-reload scenarios where the config is replaced at runtime,
// atomic.Pointer provides safe publish-and-replace without a mutex.
// Store and Load have the same memory ordering guarantees as sync/atomic.

var atomicCfg atomic.Pointer[config]

func publishConfig(c *config) {
	atomicCfg.Store(c) // atomic store: all fields visible to any Load after this
}

func readConfig() *config {
	return atomicCfg.Load() // atomic load: sees fully initialized struct or nil
}

// Ensure atomic.Pointer is used correctly (alignment check at compile time).
var _ = unsafe.Sizeof(atomicCfg)
