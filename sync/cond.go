package main

import (
	"fmt"
	"sync"
	"time"
)

// demoCondSignal shows sync.Cond with Signal: one producer wakes one waiting consumer.
//
// Cond wraps a Locker (usually a *Mutex). The pattern is always:
//
//	mu.Lock()
//	for !condition {   // loop, not if — re-check after wakeup (spurious wakeups)
//	    cond.Wait()    // atomically releases mu and suspends; re-acquires mu on wake
//	}
//	// ... use shared state ...
//	mu.Unlock()
func demoCondSignal() {
	var mu sync.Mutex
	cond := sync.NewCond(&mu)
	queue := []int{}

	// Consumer: waits until the queue has an item.
	go func() {
		mu.Lock()
		for len(queue) == 0 { // loop — not if — to handle spurious wakeups
			cond.Wait() // releases mu, parks goroutine; re-acquires mu when signalled
		}
		item := queue[0]
		queue = queue[1:]
		mu.Unlock()
		fmt.Println("  consumer: got", item)
	}()

	time.Sleep(30 * time.Millisecond)

	// Producer: adds an item and wakes one waiting consumer.
	mu.Lock()
	queue = append(queue, 42)
	cond.Signal() // wake exactly one goroutine waiting on this Cond
	mu.Unlock()
	fmt.Println("  producer: sent 42")

	time.Sleep(30 * time.Millisecond)
}

// demoCondBroadcast shows Broadcast: one event wakes ALL waiting goroutines.
// Use Broadcast when a state change is relevant to every waiter, not just one.
func demoCondBroadcast() {
	var mu sync.Mutex
	cond := sync.NewCond(&mu)
	ready := false

	var wg sync.WaitGroup

	// 4 workers all wait for the "ready" flag.
	for i := 1; i <= 4; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			mu.Lock()
			for !ready {
				cond.Wait()
			}
			mu.Unlock()
			fmt.Printf("  worker%d: starting work\n", id)
		}(i)
	}

	time.Sleep(40 * time.Millisecond)

	// Signal all workers at once (e.g. "config loaded", "server ready").
	mu.Lock()
	ready = true
	cond.Broadcast() // wake ALL waiting goroutines
	mu.Unlock()
	fmt.Println("  broadcast: ready=true")

	wg.Wait()
}
