package main

import (
	"fmt"
	"os"
	"runtime"
	"runtime/pprof"
	"sync"
	"time"
)

// demoNamedProfiles demonstrates the built-in named profiles available via
// pprof.Lookup(name). Each captures a different aspect of runtime behaviour.
//
// Available profiles:
//
//	goroutine    — stack traces of all current goroutines
//	heap         — heap allocations (same as WriteHeapProfile)
//	allocs       — all past allocations, sampled
//	block        — stack traces that led to blocking on sync primitives
//	              requires: runtime.SetBlockProfileRate(n) — n=1 captures all
//	mutex        — stack traces of holders of contended mutexes
//	              requires: runtime.SetMutexProfileFraction(n) — n=1 captures all
//	threadcreate — stack traces that led to OS thread creation

func demoNamedProfiles() {
	// ── Enable block and mutex profiling ────────────────────────────────────
	// These are off by default (rate=0) to avoid overhead.
	// Set before the code you want to profile runs.
	runtime.SetBlockProfileRate(1)        // capture every blocking event
	runtime.SetMutexProfileFraction(1)    // capture every mutex contention event
	defer runtime.SetBlockProfileRate(0)  // restore after demo
	defer runtime.SetMutexProfileFraction(0)

	// ── Generate some goroutine activity to make profiles interesting ────────
	var wg sync.WaitGroup
	ch := make(chan struct{})
	var mu sync.Mutex

	// 5 goroutines blocked on a channel (→ goroutine profile)
	for range 5 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-ch // blocked here → shows as [chan receive]
		}()
	}

	// 3 goroutines contending on a mutex (→ block + mutex profiles)
	mu.Lock()
	for range 3 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			mu.Lock() // blocked here → block profile
			mu.Unlock()
		}()
	}
	time.Sleep(30 * time.Millisecond) // let goroutines reach their blocking points

	// ── goroutine profile ────────────────────────────────────────────────────
	writeProfile("goroutine", "goroutine.prof", 0)

	// Release the mutex so the contending goroutines can proceed (feeds mutex profile)
	mu.Unlock()
	time.Sleep(10 * time.Millisecond)

	// ── block and mutex profiles ─────────────────────────────────────────────
	writeProfile("block", "block.prof", 0)
	writeProfile("mutex", "mutex.prof", 0)

	// Cleanup
	close(ch)
	wg.Wait()

	// ── All available profiles ───────────────────────────────────────────────
	fmt.Println("\n  All named profiles (pprof.Lookup):")
	for _, p := range pprof.Profiles() {
		fmt.Printf("  %-15s count=%d\n", p.Name(), p.Count())
	}

	fmt.Println()
	fmt.Println("  Inspect:")
	fmt.Println("    go tool pprof goroutine.prof")
	fmt.Println("    go tool pprof block.prof")
	fmt.Println("    go tool pprof mutex.prof")
	fmt.Println()
	fmt.Println("  Enable block/mutex profiling (off by default):")
	fmt.Println("    runtime.SetBlockProfileRate(1)      // 1 = capture every event")
	fmt.Println("    runtime.SetMutexProfileFraction(1)  // 1 = capture every contention")
}

func writeProfile(name, filename string, debug int) {
	p := pprof.Lookup(name)
	if p == nil {
		fmt.Printf("  profile %q not found\n", name)
		return
	}
	f, err := os.Create(filename)
	if err != nil {
		fmt.Printf("  error creating %s: %v\n", filename, err)
		return
	}
	defer f.Close()
	p.WriteTo(f, debug)
	fmt.Printf("  %-12s profile written → %s  (count=%d)\n", name, filename, p.Count())
}
