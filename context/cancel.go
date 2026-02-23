package main

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// demoCancel shows how to stop a goroutine on demand using WithCancel.
func demoCancel() {
	// cancel() is the only way to trigger this context.
	// Always defer it so resources are released even if we return early.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(1)

	go worker(ctx, &wg)

	time.Sleep(120 * time.Millisecond)
	fmt.Println("main: calling cancel()")
	cancel() // broadcast cancellation to all goroutines watching this ctx

	wg.Wait()
	fmt.Println("main: worker stopped, ctx.Err():", ctx.Err()) // context.Canceled
}

func worker(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	tick := time.NewTicker(40 * time.Millisecond)
	defer tick.Stop()

	for {
		select {
		case <-tick.C:
			fmt.Println("worker: tick")
		case <-ctx.Done():
			// ctx.Done() is closed when cancel() is called (or deadline/timeout fires).
			fmt.Println("worker: done, reason:", ctx.Err())
			return
		}
	}
}
