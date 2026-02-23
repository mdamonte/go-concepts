package main

import (
	"fmt"
	"sync"
	"time"
)

// demoDone shows the done-channel pattern: a channel used purely as a
// broadcast signal with no data. Closing it wakes every goroutine that
// is blocked on a receive from it — unlike sending a value, which wakes
// only one goroutine.
//
// This is the foundation of context.Context cancellation.
func demoDone() {
	done := make(chan struct{})

	var wg sync.WaitGroup
	for i := 1; i <= 3; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			fmt.Printf("  goroutine%d: waiting\n", id)
			<-done // all three block here
			fmt.Printf("  goroutine%d: done signal received\n", id)
		}(i)
	}

	time.Sleep(60 * time.Millisecond)
	fmt.Println("  broadcasting done...")
	close(done) // wakes ALL blocked goroutines simultaneously

	wg.Wait()
}

// demoOrDone shows the or-done wrapper: read from a value channel but
// stop immediately if a done signal arrives. Avoids goroutine leaks when
// the producer outlives the consumer.
func demoOrDone() {
	done := make(chan struct{})
	values := make(chan int)

	// Producer sends indefinitely.
	go func() {
		for i := 0; ; i++ {
			select {
			case <-done:
				return
			case values <- i:
			}
		}
	}()

	// Consumer reads a few values, then cancels.
	for v := range orDone(done, values) {
		fmt.Printf("  %d ", v)
		if v >= 4 {
			close(done) // signal producer to stop
			break
		}
	}
	fmt.Println()
}

// orDone wraps a value channel so that ranging over the returned channel
// is always safe: it exits cleanly when done is closed, even if the
// underlying channel is never closed by its producer.
func orDone(done <-chan struct{}, in <-chan int) <-chan int {
	out := make(chan int)
	go func() {
		defer close(out)
		for {
			select {
			case <-done:
				return
			case v, ok := <-in:
				if !ok {
					return
				}
				select {
				case out <- v:
				case <-done:
					return
				}
			}
		}
	}()
	return out
}
