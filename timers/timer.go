package main

import (
	"fmt"
	"time"
)

// demoTimer shows the basic Timer lifecycle:
// NewTimer → wait on .C → fires once after the duration.
func demoTimer() {
	timer := time.NewTimer(80 * time.Millisecond)

	fmt.Println("  waiting for timer...")
	t := <-timer.C // blocks until the timer fires
	fmt.Printf("  fired at %s\n", t.Format("15:04:05.000"))
}

// demoTimerStop shows how to cancel a timer that hasn't fired yet.
//
// Stop returns false if the timer has already fired or been stopped.
// When Stop returns false, the value may already be in .C — you must
// drain the channel to avoid a later receive seeing a stale tick.
//
// Correct stop-and-drain pattern (Go < 1.23):
//
//	if !timer.Stop() {
//	    <-timer.C
//	}
//
// Go 1.23 simplified this: Reset no longer requires a prior drain, but
// the pattern above is still correct and safe on all versions.
func demoTimerStop() {
	timer := time.NewTimer(200 * time.Millisecond)

	// Cancel before it fires.
	stopped := timer.Stop()
	fmt.Printf("  Stop() returned: %v (true = cancelled in time)\n", stopped)

	if !stopped {
		// Drain to avoid a ghost tick reaching a future select.
		<-timer.C
		fmt.Println("  drained ghost tick")
	}

	// Confirm the channel is empty — no tick arrives after 300 ms.
	select {
	case <-timer.C:
		fmt.Println("  unexpected tick")
	case <-time.After(300 * time.Millisecond):
		fmt.Println("  confirmed: no tick after Stop()")
	}
}

// demoTimerReset shows how to reuse a timer for a new duration.
//
// Safe reset sequence (Go < 1.23):
//  1. Stop the timer.
//  2. Drain .C if Stop returned false (timer had already fired).
//  3. Call Reset.
func demoTimerReset() {
	timer := time.NewTimer(500 * time.Millisecond)

	// Stop and drain before resetting to avoid a stale tick.
	if !timer.Stop() {
		select {
		case <-timer.C:
		default:
		}
	}
	timer.Reset(60 * time.Millisecond) // new shorter duration

	t := <-timer.C
	fmt.Printf("  reset timer fired at %s\n", t.Format("15:04:05.000"))
}

// demoAfterFunc shows time.AfterFunc: calls a function in its own goroutine
// after the duration. Useful for background callbacks without a channel.
//
// The returned *Timer can still be stopped with Stop().
func demoAfterFunc() {
	done := make(chan struct{})

	t := time.AfterFunc(60 * time.Millisecond, func() {
		fmt.Printf("  AfterFunc callback at %s\n", time.Now().Format("15:04:05.000"))
		close(done)
	})

	fmt.Printf("  AfterFunc scheduled at %s\n", time.Now().Format("15:04:05.000"))
	<-done

	// Stopping after the callback has already run is a safe no-op.
	t.Stop()
}
