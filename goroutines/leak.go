package main

import (
	"context"
	"fmt"
	"runtime"
	"time"
)

// A goroutine leak occurs when a goroutine is created but can never exit.
// The goroutine stays alive (and holds its memory) for the duration of the
// program. Leaks are invisible at runtime unless you instrument NumGoroutine.
//
// Common causes:
//   1. Blocked forever waiting to receive from a channel nobody writes to.
//   2. Blocked forever waiting to send to a channel nobody reads from.
//   3. Infinite loop with no exit condition and no cancellation signal.

// demoLeakSend shows a goroutine that leaks because it tries to send on a
// channel that the main goroutine is no longer reading.
func demoLeakSend() {
	before := runtime.NumGoroutine()

	leak := func() {
		ch := make(chan int) // unbuffered; nobody will read after this returns
		go func() {
			fmt.Println("  leaking goroutine: trying to send...")
			ch <- 42 // blocks forever — LEAK
		}()
		// function returns; ch goes out of scope but the goroutine is stuck
	}

	leak()
	time.Sleep(20 * time.Millisecond) // let the goroutine reach the send
	fmt.Printf("  goroutines before: %d  after leak: %d  (delta: +%d)\n",
		before, runtime.NumGoroutine(), runtime.NumGoroutine()-before)
	// The leaked goroutine will stay until the program exits.
}

// demoLeakReceive shows a goroutine that leaks because it waits forever on a
// channel that the producer stops writing to without closing.
func demoLeakReceive() {
	before := runtime.NumGoroutine()

	leak := func() {
		ch := make(chan int)
		go func() {
			fmt.Println("  leaking goroutine: waiting to receive...")
			<-ch // blocks forever — nobody will send — LEAK
		}()
		// Forgot to close(ch) or send a value; goroutine is stuck.
	}

	leak()
	time.Sleep(20 * time.Millisecond)
	fmt.Printf("  goroutines before: %d  after leak: %d  (delta: +%d)\n",
		before, runtime.NumGoroutine(), runtime.NumGoroutine()-before)
}

// demoLeakFixed shows both fixes applied together using context cancellation.
//
// Fix for send leak:   use a buffered channel so the send doesn't block, or
//                      listen to ctx.Done() in a select.
// Fix for receive leak: listen to ctx.Done() in a select so the goroutine can
//                      exit when the caller is done.
func demoLeakFixed() {
	before := runtime.NumGoroutine()
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// Fixed send: goroutine exits via ctx.Done() if nobody reads.
	fixedSend := func(ctx context.Context) {
		ch := make(chan int, 1) // buffered: send won't block if buffer has space
		go func() {
			select {
			case ch <- 42:
				fmt.Println("  fixed send: value sent")
			case <-ctx.Done():
				fmt.Println("  fixed send: context cancelled, goroutine exiting")
			}
		}()
	}

	// Fixed receive: goroutine exits via ctx.Done() if nobody sends.
	fixedReceive := func(ctx context.Context) {
		ch := make(chan int)
		go func() {
			select {
			case v := <-ch:
				fmt.Println("  fixed receive: got", v)
			case <-ctx.Done():
				fmt.Println("  fixed receive: context cancelled, goroutine exiting")
			}
		}()
	}

	fixedSend(ctx)
	fixedReceive(ctx)

	<-ctx.Done() // wait for timeout
	time.Sleep(10 * time.Millisecond)
	fmt.Printf("  goroutines before: %d  after: %d  (delta: %d)\n",
		before, runtime.NumGoroutine(), runtime.NumGoroutine()-before)
}
