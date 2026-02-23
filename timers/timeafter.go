package main

import (
	"fmt"
	"time"
)

// demoTimeAfter shows time.After: returns a channel that receives the
// current time after the given duration. Shorthand for:
//
//	time.NewTimer(d).C
//
// The underlying Timer is released after the channel fires.
// However, if the channel is never received (e.g. the select takes another
// branch first), the Timer is NOT garbage-collected until it fires —
// this can hold memory for the full duration in tight loops.
func demoTimeAfter() {
	fmt.Println("  waiting 60 ms via time.After...")
	t := <-time.After(60 * time.Millisecond)
	fmt.Printf("  received at %s\n", t.Format("15:04:05.000"))
}

// demoTimeout is the canonical use of time.After in a select:
// race a slow operation against a deadline.
//
// ⚠ Leak risk in loops: each iteration creates a new Timer that is only
// released after it fires — even if the other case won.
// For loop-based timeouts, use time.NewTimer and reuse it.
func demoTimeout() {
	result := make(chan string)

	go func() {
		time.Sleep(200 * time.Millisecond) // slow operation
		result <- "data"
	}()

	select {
	case v := <-result:
		fmt.Println("  got result:", v)
	case <-time.After(100 * time.Millisecond):
		fmt.Println("  timed out waiting for result")
	}
}

// demoTimeAfterLeak demonstrates the memory-leak risk when time.After is
// used inside a loop. Each call creates a Timer that lives until it fires,
// regardless of whether its channel was ever read.
//
// Rule: in a hot loop, replace time.After with time.NewTimer + Reset.
func demoTimeAfterLeak() {
	// ── Leaky pattern (shown, not run in a tight loop) ───────────────────────
	//
	// for {
	//     select {
	//     case msg := <-messages:
	//         process(msg)
	//     case <-time.After(5 * time.Second): // new Timer every iteration!
	//         fmt.Println("idle timeout")
	//         return
	//     }
	// }

	// ── Correct pattern: reuse a single Timer ────────────────────────────────
	messages := make(chan string, 3)
	messages <- "a"
	messages <- "b"
	messages <- "c"
	close(messages)

	idleTimeout := time.NewTimer(200 * time.Millisecond)
	defer idleTimeout.Stop()

	for {
		// Reset the idle timer on every iteration to extend the deadline.
		if !idleTimeout.Stop() {
			select {
			case <-idleTimeout.C:
			default:
			}
		}
		idleTimeout.Reset(200 * time.Millisecond)

		select {
		case msg, ok := <-messages:
			if !ok {
				fmt.Println("  channel closed, done")
				return
			}
			fmt.Println("  message:", msg)
		case <-idleTimeout.C:
			fmt.Println("  idle timeout")
			return
		}
	}
}
