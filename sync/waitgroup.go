package main

import (
	"fmt"
	"sync"
	"time"
)

// demoWaitGroup shows the canonical fan-out-then-wait pattern.
//
// Rules:
//   - Call wg.Add before launching the goroutine (not inside it) to avoid
//     a race between Add and Wait.
//   - Call wg.Done via defer so it runs even if the goroutine panics.
//   - Call wg.Wait in the goroutine that owns the group (usually main or
//     the goroutine that launched the workers).
func demoWaitGroup() {
	var wg sync.WaitGroup

	for i := 1; i <= 5; i++ {
		wg.Add(1) // increment BEFORE launching
		go func(id int) {
			defer wg.Done() // decrement when goroutine exits
			time.Sleep(time.Duration(id) * 10 * time.Millisecond)
			fmt.Printf("  worker%d done\n", id)
		}(i)
	}

	wg.Wait() // blocks until the counter reaches zero
	fmt.Println("all workers finished")
}
