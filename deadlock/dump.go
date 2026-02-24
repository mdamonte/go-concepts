package main

import (
	"fmt"
	"runtime"
	"strings"
)

// dumpGoroutines captures a full goroutine snapshot via runtime.Stack and
// prints each goroutine with its blocking state label and top stack frames.
//
// This is the same output the runtime produces on:
//   fatal error: all goroutines are asleep - deadlock!
//
// Key state labels to recognise:
//
//	[running]       — goroutine is currently executing on an OS thread
//	[runnable]      — ready to run, waiting for a free OS thread
//	[chan receive]  — blocked on <-ch
//	[chan send]     — blocked on ch <- v
//	[select]        — blocked in select, all cases are blocking
//	[semacquire]    — blocked on sync.Mutex.Lock / sync.RWMutex / semaphore
//	[IO wait]       — blocked on network/file I/O (poll.runtime_pollWait)
//	[sleep]         — inside time.Sleep
//	[syscall]       — executing a blocking OS syscall
func dumpGoroutines() {
	buf := make([]byte, 256*1024)
	n := runtime.Stack(buf, true)
	raw := strings.TrimSpace(string(buf[:n]))

	fmt.Println()
	for _, block := range strings.Split(raw, "\n\n") {
		block = strings.TrimSpace(block)
		if block == "" {
			continue
		}
		lines := strings.Split(block, "\n")

		// Line 0: "goroutine N [state, X minutes]:"
		fmt.Printf("  %s\n", lines[0])

		// Show up to 4 stack frames for context.
		limit := min(len(lines)-1, 4)
		for i := 1; i <= limit; i++ {
			fmt.Printf("  %s\n", lines[i])
		}
		if len(lines)-1 > limit {
			fmt.Printf("  ... (+%d lines)\n", len(lines)-1-limit)
		}
		fmt.Println()
	}
}
