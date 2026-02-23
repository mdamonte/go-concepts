package main

import (
	"fmt"
	"time"
)

// demoSelect shows how select multiplexes over multiple channel operations.
// On each iteration Go evaluates all cases; if more than one is ready it
// picks one at random (fair, non-deterministic).
func demoSelect() {
	ch1 := make(chan string, 1)
	ch2 := make(chan string, 1)

	ch1 <- "one"
	ch2 <- "two"

	// Both channels are ready; Go picks randomly.
	for i := 0; i < 2; i++ {
		select {
		case v := <-ch1:
			fmt.Println("received from ch1:", v)
		case v := <-ch2:
			fmt.Println("received from ch2:", v)
		}
	}
}

// demoSelectDefault shows the default case: it runs immediately when no
// other case is ready, making the select non-blocking.
// Useful for try-send / try-receive without goroutines.
func demoSelectDefault() {
	ch := make(chan int, 1)

	// Try-send: send only if the channel has space right now.
	select {
	case ch <- 10:
		fmt.Println("sent 10")
	default:
		fmt.Println("channel full, skipped send")
	}

	// Try-receive: receive only if a value is available right now.
	select {
	case v := <-ch:
		fmt.Println("received:", v)
	default:
		fmt.Println("nothing to receive")
	}

	// Try-receive on empty channel → default fires.
	select {
	case v := <-ch:
		fmt.Println("received:", v)
	default:
		fmt.Println("channel empty, nothing received")
	}
}

// demoSelectNil shows a key property: a nil channel is never ready.
// A case on a nil channel is permanently skipped by the scheduler.
//
// This is useful to dynamically disable a case inside a select loop
// without restructuring the whole select block.
func demoSelectNil() {
	var disabled chan string // nil
	active := make(chan string, 1)
	active <- "hello"

	select {
	case v := <-disabled: // never selected
		fmt.Println("disabled:", v)
	case v := <-active:
		fmt.Println("active:", v)
	}

	// Practical pattern: disable a case once it's been handled.
	a := make(chan int, 1)
	b := make(chan int, 1)
	a <- 1
	b <- 2

	for i := 0; i < 2; i++ {
		select {
		case v, ok := <-a:
			if !ok {
				a = nil // set to nil to disable this case
				continue
			}
			fmt.Println("a:", v)
			a = nil // disable after first receive
		case v, ok := <-b:
			if !ok {
				b = nil
				continue
			}
			fmt.Println("b:", v)
			b = nil
		}
	}
}

// demoSelectTimeout shows the canonical timeout pattern:
// race a channel receive against time.After.
func demoSelectTimeout() {
	slow := make(chan string)

	go func() {
		time.Sleep(200 * time.Millisecond)
		slow <- "result"
	}()

	select {
	case v := <-slow:
		fmt.Println("got:", v)
	case <-time.After(100 * time.Millisecond):
		fmt.Println("timeout: slow channel didn't respond in time")
	}
}
