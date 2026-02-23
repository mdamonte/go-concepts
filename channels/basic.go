package main

import (
	"fmt"
	"sync"
)

// demoUnbuffered shows that an unbuffered channel has zero capacity:
// the sender blocks until a receiver is ready, and vice versa.
// This guarantees synchronization between two goroutines.
func demoUnbuffered() {
	ch := make(chan int) // capacity = 0

	go func() {
		fmt.Println("sender: sending 42")
		ch <- 42 // blocks here until someone receives
		fmt.Println("sender: send complete")
	}()

	v := <-ch // blocks here until someone sends
	fmt.Println("receiver: got", v)
}

// demoBuffered shows that a buffered channel decouples sender and receiver:
// the sender only blocks when the buffer is full; the receiver only blocks
// when the buffer is empty.
func demoBuffered() {
	ch := make(chan string, 3) // capacity = 3

	// These three sends don't block because the buffer can absorb them.
	ch <- "a"
	ch <- "b"
	ch <- "c"
	fmt.Println("sent 3 items without blocking, len:", len(ch), "cap:", cap(ch))

	// Receive all three without launching a goroutine.
	fmt.Println(<-ch, <-ch, <-ch)
}

// demoDirectional shows channel direction types.
// send-only  chan<- T   receive-only  <-chan T
//
// Direction is enforced by the compiler: trying to receive from a
// send-only channel (or vice versa) is a compile-time error.
// Use directional types in function signatures to document intent clearly.
func demoDirectional() {
	ch := make(chan int, 1)
	produce(ch) // accepts chan<- int
	consume(ch) // accepts <-chan int
}

func produce(out chan<- int) {
	out <- 99
	fmt.Println("produce: sent 99")
}

func consume(in <-chan int) {
	fmt.Println("consume: got", <-in)
}

// demoCloseRange shows how to signal that no more values will be sent.
//
// Rules:
//   - Only the sender should close a channel; closing from the receiver side
//     is a design smell.
//   - Receiving from a closed channel returns the zero value immediately.
//   - Use the comma-ok idiom (v, ok := <-ch) to distinguish zero values
//     from a closed channel.
//   - range over a channel loops until the channel is closed.
func demoCloseRange() {
	ch := make(chan int, 5)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 1; i <= 5; i++ {
			ch <- i
		}
		close(ch) // signals receivers: no more values coming
	}()

	// range exits automatically when ch is closed and drained.
	for v := range ch {
		fmt.Print(v, " ")
	}
	fmt.Println()
	wg.Wait()

	// Receiving from a closed, empty channel: zero value + ok=false.
	v, ok := <-ch
	fmt.Printf("after close: v=%d ok=%v\n", v, ok)
}
