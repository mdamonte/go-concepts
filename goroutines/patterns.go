package main

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// demoFireAndForget shows a goroutine that runs a background task with no
// return value and no coordination — the caller doesn't wait for it.
// Use for non-critical background work (e.g. async logging, metrics).
// Always give it a way to exit (done channel or context) to avoid leaks.
func demoFireAndForget() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Background heartbeat — fires and is forgotten by main.
	go func(ctx context.Context) {
		tick := time.NewTicker(20 * time.Millisecond)
		defer tick.Stop()
		for {
			select {
			case <-tick.C:
				fmt.Println("  heartbeat tick")
			case <-ctx.Done():
				fmt.Println("  heartbeat stopped")
				return
			}
		}
	}(ctx)

	time.Sleep(70 * time.Millisecond)
	cancel() // stop the background goroutine
	time.Sleep(10 * time.Millisecond)
}

// demoFirstWins launches N goroutines doing the same task and returns the
// result of whichever finishes first. Remaining goroutines are cancelled via
// context so they don't leak.
//
// Use case: querying multiple replicas, redundant API calls, hedged requests.
func demoFirstWins() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	type response struct {
		worker int
		value  string
	}

	ch := make(chan response, 3) // buffered so slow goroutines can still send and exit

	latencies := []time.Duration{60, 20, 40} // worker2 wins
	for i, lat := range latencies {
		go func(id int, latency time.Duration) {
			select {
			case <-time.After(latency):
				ch <- response{worker: id, value: fmt.Sprintf("result-from-worker%d", id)}
			case <-ctx.Done():
				// context was cancelled before we finished; exit cleanly
			}
		}(i+1, lat)
	}

	first := <-ch
	cancel() // cancel the remaining goroutines
	fmt.Printf("  first response: worker%d → %s\n", first.worker, first.value)
}

// demoBounded launches many goroutines but limits how many run concurrently
// using a semaphore channel. Prevents thundering herd and resource exhaustion.
func demoBounded() {
	const total = 12
	const maxConcurrent = 3

	sem := make(chan struct{}, maxConcurrent)
	var wg sync.WaitGroup
	var mu sync.Mutex
	peak := 0
	running := 0

	for i := 1; i <= total; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			sem <- struct{}{}        // acquire
			defer func() { <-sem }() // release

			mu.Lock()
			running++
			if running > peak {
				peak = running
			}
			mu.Unlock()

			fmt.Printf("  task%02d running\n", id)
			time.Sleep(15 * time.Millisecond)

			mu.Lock()
			running--
			mu.Unlock()
		}(i)
	}

	wg.Wait()
	fmt.Printf("  peak concurrency: %d (max allowed: %d)\n", peak, maxConcurrent)
}
