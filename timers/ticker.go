package main

import (
	"fmt"
	"time"
)

// demoTicker shows the basic Ticker lifecycle.
//
// A Ticker fires on .C repeatedly at the given interval.
// ALWAYS call Stop() when done — a Ticker that is never stopped keeps a
// goroutine and a channel alive for the lifetime of the program (leak).
func demoTicker() {
	ticker := time.NewTicker(40 * time.Millisecond)
	defer ticker.Stop() // critical: free the internal goroutine

	deadline := time.After(160 * time.Millisecond)

	for {
		select {
		case t := <-ticker.C:
			fmt.Printf("  tick at %s\n", t.Format("15:04:05.000"))
		case <-deadline:
			fmt.Println("  deadline reached, stopping ticker")
			return
		}
	}
}

// demoTickerReset shows how to change a running ticker's interval on the fly.
//
// Reset stops the current ticker and starts it anew with the new duration.
// The channel is NOT drained automatically — ticks sent before Reset was
// called may still be in the buffer. Read and discard them if needed.
func demoTickerReset() {
	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()

	fmt.Println("  phase 1: 20 ms interval")
	for i := 0; i < 3; i++ {
		t := <-ticker.C
		fmt.Printf("    tick %d at %s\n", i+1, t.Format("15:04:05.000"))
	}

	ticker.Reset(70 * time.Millisecond) // switch to a slower interval
	fmt.Println("  phase 2: 70 ms interval")
	for i := 0; i < 3; i++ {
		t := <-ticker.C
		fmt.Printf("    tick %d at %s\n", i+1, t.Format("15:04:05.000"))
	}
}

// demoTimeTick shows time.Tick: a convenience wrapper that returns just
// the channel without exposing the underlying Ticker.
//
// WARNING: the Ticker is never garbage-collected because there is no way
// to call Stop() on it. Only use time.Tick at the top-level of a long-lived
// program (e.g. main loop). Never call it inside a function or a goroutine
// that may be called more than once — each call leaks a Ticker.
func demoTimeTick() {
	fmt.Println("  time.Tick is safe here because main() runs only once")

	ch := time.Tick(50 * time.Millisecond) // leaks if called repeatedly
	deadline := time.After(160 * time.Millisecond)

	for {
		select {
		case t := <-ch:
			fmt.Printf("  time.Tick at %s\n", t.Format("15:04:05.000"))
		case <-deadline:
			fmt.Println("  done")
			return
		}
	}
}
