package main

import (
	"fmt"
	"sync"
)

// demoMutex shows how sync.Mutex protects a shared variable from concurrent writes.
//
// Without the mutex the goroutines would race on `counter`, producing
// non-deterministic results (run with -race to detect it).
// With the mutex, only one goroutine can be inside the critical section at a time.
func demoMutex() {
	var (
		mu      sync.Mutex
		counter int
		wg      sync.WaitGroup
	)

	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			mu.Lock()
			defer mu.Unlock() // idiomatic: release on every return path
			counter++
		}()
	}

	wg.Wait()
	fmt.Println("counter:", counter) // always 1000
}

// demoRWMutex shows sync.RWMutex: multiple goroutines can hold a read lock
// simultaneously, but a write lock is exclusive.
//
// Use RWMutex when reads are frequent and writes are rare — it avoids
// unnecessary serialization between concurrent readers.
func demoRWMutex() {
	var (
		mu    sync.RWMutex
		cache = map[string]string{"lang": "Go"}
		wg    sync.WaitGroup
	)

	// 5 concurrent readers — all acquire RLock simultaneously.
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			mu.RLock()
			defer mu.RUnlock()
			fmt.Printf("  reader%d: lang=%s\n", id, cache["lang"])
		}(i)
	}

	// 1 writer — waits until all current readers release, then takes exclusive lock.
	wg.Add(1)
	go func() {
		defer wg.Done()
		mu.Lock()
		defer mu.Unlock()
		cache["lang"] = "Go 1.21"
		fmt.Println("  writer:  updated lang")
	}()

	wg.Wait()
}
