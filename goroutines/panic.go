package main

import (
	"fmt"
	"sync"
	"time"
)

// demoPanic shows two critical rules about panics and goroutines:
//
//  1. A panic in a goroutine crashes the entire program, not just that goroutine.
//     Unlike errors, a panic cannot be "caught" from outside the goroutine.
//
//  2. recover() only works inside the same goroutine where the panic occurred,
//     and only when called directly inside a deferred function.
func demoPanic() {
	demoPanicRecoverSameGoroutine()
	demoPanicRecoverWrapper()
	demoPanicInGoroutine()
}

// demoPanicRecoverSameGoroutine shows the basic pattern: defer + recover
// in the same goroutine that may panic.
func demoPanicRecoverSameGoroutine() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("  recovered in same goroutine:", r)
		}
	}()

	fmt.Println("  about to panic...")
	panic("something went wrong") // caught by the deferred recover above
}

// demoPanicRecoverWrapper shows the safeGo pattern: a wrapper that launches
// a goroutine and recovers any panic, forwarding it as an error via a channel.
// Use this in production when a goroutine crash would take down a server.
func demoPanicRecoverWrapper() {
	var wg sync.WaitGroup
	errs := make(chan error, 3)

	for i := 1; i <= 3; i++ {
		wg.Add(1)
		safeGo(&wg, errs, func(id int) func() {
			return func() {
				if id == 2 {
					panic(fmt.Sprintf("goroutine%d exploded", id))
				}
				fmt.Printf("  safeGo: goroutine%d finished ok\n", id)
			}
		}(i))
	}

	wg.Wait()
	close(errs)
	for err := range errs {
		fmt.Println("  safeGo caught:", err)
	}
}

// safeGo launches fn in a goroutine. If fn panics, the panic is recovered
// and the resulting error is sent to errs instead of crashing the program.
func safeGo(wg *sync.WaitGroup, errs chan<- error, fn func()) {
	wg.Add(0) // already added by caller
	go func() {
		defer wg.Done()
		defer func() {
			if r := recover(); r != nil {
				errs <- fmt.Errorf("panic: %v", r)
			}
		}()
		fn()
	}()
}

// demoPanicInGoroutine demonstrates that recover() in main CANNOT catch a
// panic from a different goroutine. The only way to survive is to recover
// inside that goroutine itself.
//
// This example uses a goroutine that recovers its own panic to avoid crashing
// the demo; in real code a goroutine with an unrecovered panic kills the process.
func demoPanicInGoroutine() {
	done := make(chan struct{})

	go func() {
		defer close(done)
		defer func() {
			if r := recover(); r != nil {
				// Must recover here — recovery in main() would be too late.
				fmt.Println("  goroutine recovered its own panic:", r)
			}
		}()

		time.Sleep(10 * time.Millisecond)
		panic("goroutine-level panic")
	}()

	<-done
	fmt.Println("  main: goroutine finished (panic was handled inside it)")
}
