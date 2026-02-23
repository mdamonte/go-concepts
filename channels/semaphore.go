package main

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// demoSemaphore shows how a buffered channel acts as a counting semaphore:
// at most N goroutines run concurrently at any time.
//
//	Acquire: sem <- struct{}{}  (blocks when buffer is full)
//	Release: <-sem              (always in a defer)
//
// This is simpler and more composable than sync.Mutex for rate-limiting
// concurrency without a worker pool.
func demoSemaphore() {
	const maxConcurrent = 3
	const totalTasks = 9

	sem := make(chan struct{}, maxConcurrent)

	var (
		wg      sync.WaitGroup
		running atomic.Int32
	)

	for i := 1; i <= totalTasks; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			sem <- struct{}{} // acquire: blocks if maxConcurrent goroutines are running
			defer func() { <-sem }() // release

			n := running.Add(1)
			fmt.Printf("  task%d started  (concurrent: %d)\n", id, n)
			time.Sleep(30 * time.Millisecond)
			running.Add(-1)
			fmt.Printf("  task%d finished\n", id)
		}(i)
	}

	wg.Wait()
}
