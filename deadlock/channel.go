package main

import (
	"fmt"
	"time"
)

// demoChanReceive shows goroutine state [chan receive]:
// the goroutine is blocked on a direct receive from an unbuffered channel
// that has no sender.
//
// Goroutine dump entry:
//
//	goroutine N [chan receive]:
//	main.demoChanReceive.func1()
//	    .../channel.go:NN +0x...
func demoChanReceive() {
	ch := make(chan int) // unbuffered — no sender will ever write

	go func() {
		fmt.Println("  goroutine: blocking on  v := <-ch  (no sender)")
		v := <-ch // ← blocked here, shows as [chan receive]
		fmt.Println("  goroutine: received", v) // unreachable
	}()

	time.Sleep(80 * time.Millisecond) // let the goroutine reach the block
	dumpGoroutines()
	// goroutine is intentionally leaked — it will appear in the final crash dump
}

// demoChanSend shows goroutine state [chan send]:
// the goroutine is blocked trying to send to an unbuffered channel
// that has no receiver.
//
// Goroutine dump entry:
//
//	goroutine N [chan send]:
//	main.demoChanSend.func1()
//	    .../channel.go:NN +0x...
func demoChanSend() {
	ch := make(chan int) // unbuffered — no receiver will ever read

	go func() {
		fmt.Println("  goroutine: blocking on  ch <- 42  (no receiver)")
		ch <- 42 // ← blocked here, shows as [chan send]
		fmt.Println("  goroutine: sent (unreachable)")
	}()

	time.Sleep(80 * time.Millisecond)
	dumpGoroutines()
	// goroutine intentionally leaked
}

// demoSelect shows goroutine state [select]:
// the goroutine is blocked in a select where every case is itself blocked.
// The runtime uses [select] instead of [chan receive] when the goroutine
// entered via a select statement.
//
// Goroutine dump entry:
//
//	goroutine N [select]:
//	main.demoSelect.func1()
//	    .../channel.go:NN +0x...
func demoSelect() {
	ch1 := make(chan int)    // no sender
	ch2 := make(chan string) // no sender

	go func() {
		fmt.Println("  goroutine: blocking in select (both channels empty, no senders)")
		select {
		case v := <-ch1: // ← blocked
			fmt.Println("  goroutine: got int", v)
		case s := <-ch2: // ← blocked
			fmt.Println("  goroutine: got string", s)
		}
		// shows as [select], not [chan receive]
	}()

	time.Sleep(80 * time.Millisecond)
	dumpGoroutines()
	// goroutine intentionally leaked
}
