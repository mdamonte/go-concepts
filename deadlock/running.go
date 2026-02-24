package main

import (
	"fmt"
	"runtime"
	"time"
)

// demoRunning shows goroutine state [running]:
//
//   - The goroutine that calls runtime.Stack always appears as [running]
//     in its own dump entry — it is the currently executing goroutine.
//
//   - A CPU-spinning goroutine shows as [running] when it happens to be
//     on an OS thread at snapshot time, or [runnable] when it has been
//     preempted between iterations.
//
// [running] is NOT a blocking state — you see it in pprof CPU profiles
// as a hotspot and in goroutine dumps as the goroutine that triggered
// the dump. In a pure deadlock dump every goroutine would be [blocked];
// if you see [running] there it is the runtime itself printing the crash.
//
// Goroutine dump entries:
//
//	goroutine 1 [running]:          ← the goroutine calling dumpGoroutines()
//	goroutine N [running]:          ← the busy loop goroutine, if on-CPU now
//	  OR
//	goroutine N [runnable]:         ← same goroutine, preempted at snapshot
func demoRunning() {
	stop := make(chan struct{})
	done := make(chan struct{})

	go func() {
		defer close(done)
		fmt.Println("  goroutine: busy loop — appears as [running] or [runnable]")
		i := 0
		for {
			select {
			case <-stop:
				fmt.Printf("  goroutine: stopped after %d iterations\n", i)
				return
			default:
				i++
				// Gosched yields to other goroutines so the loop does not
				// starve them. At the moment of the snapshot this goroutine
				// may or may not hold an OS thread — hence [running] vs [runnable].
				runtime.Gosched()
			}
		}
	}()

	time.Sleep(80 * time.Millisecond)
	// dumpGoroutines itself shows [running] for the calling goroutine.
	dumpGoroutines()

	// IMPORTANT: wait for the goroutine to fully exit before continuing.
	// If it is still [runnable] when the final deadlock demo runs, the
	// runtime deadlock detector will not fire (a runnable goroutine means
	// the program can still make progress).
	close(stop)
	<-done
}
