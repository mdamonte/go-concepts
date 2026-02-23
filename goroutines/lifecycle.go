package main

import (
	"fmt"
	"runtime"
	"sync"
)

// demoLifecycle illustrates key runtime properties of goroutines.
func demoLifecycle() {
	demoGOMAXPROCS()
	demoNumGoroutine()
	demoGosched()
	demoStackGrowth()
}

// demoGOMAXPROCS shows how Go maps goroutines onto OS threads.
//
// Go uses an M:N scheduler: M goroutines are multiplexed over N OS threads (P).
// GOMAXPROCS controls how many OS threads run Go code simultaneously.
// Default is runtime.NumCPU() since Go 1.5.
func demoGOMAXPROCS() {
	prev := runtime.GOMAXPROCS(0) // 0 = query without changing
	fmt.Printf("  GOMAXPROCS: %d  (NumCPU: %d)\n", prev, runtime.NumCPU())

	// Temporarily limit to 1 OS thread.
	runtime.GOMAXPROCS(1)
	fmt.Printf("  set to 1, now: %d\n", runtime.GOMAXPROCS(0))
	runtime.GOMAXPROCS(prev) // restore
}

// demoNumGoroutine shows how to observe goroutine count at runtime.
// Useful to detect leaks: if the count keeps growing, goroutines aren't exiting.
func demoNumGoroutine() {
	before := runtime.NumGoroutine()
	fmt.Printf("  goroutines before: %d\n", before)

	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// just exist for a moment
		}()
	}

	// Goroutines may not all be scheduled yet; count can vary.
	fmt.Printf("  goroutines during: %d\n", runtime.NumGoroutine())
	wg.Wait()
	fmt.Printf("  goroutines after:  %d\n", runtime.NumGoroutine())
}

// demoGosched shows runtime.Gosched(): voluntarily yield the current goroutine's
// time slice, allowing other goroutines to run on the same OS thread.
//
// Rarely needed in production — the scheduler is preemptive since Go 1.14.
// Useful in tight CPU-bound loops that never block, or in tests.
func demoGosched() {
	var wg sync.WaitGroup
	printed := make(chan string, 10)

	for i := 1; i <= 3; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			printed <- fmt.Sprintf("goroutine%d: before yield", id)
			runtime.Gosched() // yield: let others run
			printed <- fmt.Sprintf("goroutine%d: after yield", id)
		}(i)
	}

	wg.Wait()
	close(printed)
	for msg := range printed {
		fmt.Println(" ", msg)
	}
}

// demoStackGrowth shows that goroutine stacks start tiny (~2–8 KB) and grow
// automatically. This is why Go can run millions of goroutines while threads
// require a fixed-size stack (typically 1–8 MB each).
func demoStackGrowth() {
	done := make(chan int)
	go func() {
		// Each recursive call adds a frame; Go grows the stack transparently.
		done <- deepRecurse(10000)
	}()
	fmt.Printf("  deep recursion result: %d\n", <-done)
}

func deepRecurse(n int) int {
	if n == 0 {
		return 0
	}
	return 1 + deepRecurse(n-1) // triggers stack growth
}
