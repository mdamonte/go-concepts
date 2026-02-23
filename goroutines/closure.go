package main

import (
	"fmt"
	"sync"
)

// demoClosure shows the classic loop-variable capture bug and its two fixes.
//
// The bug: a goroutine closure captures the *variable* i, not its value at
// launch time. By the time the goroutines run, the loop may have finished and
// i holds its final value — all goroutines print the same number.
//
// Note: Go 1.22 changed loop semantics so each iteration has its own variable,
// making fix 2 unnecessary on modern toolchains. Fix 1 (argument passing) still
// documents intent clearly and works on all versions.
func demoClosure() {
	var wg sync.WaitGroup

	// ── BUG (illustrative — run with go1.21 or earlier to observe) ──────────
	// Each goroutine captures the address of i. When they execute, i is
	// already 5. All goroutines may print "5".
	//
	// buggy := make(chan int, 5)
	// for i := 0; i < 5; i++ {
	//     go func() { buggy <- i }()  // ← captures &i, not the value
	// }

	// ── Fix 1: pass i as an argument (works on all Go versions) ─────────────
	// The value of i is copied into the parameter n at call time.
	fmt.Println("  fix 1 — pass as argument:")
	results := make(chan int, 5)
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(n int) { // n is a fresh copy of i for each iteration
			defer wg.Done()
			results <- n
		}(i) // i evaluated here, at launch time
	}
	wg.Wait()
	close(results)
	for v := range results {
		fmt.Printf("  %d", v)
	}
	fmt.Println()

	// ── Fix 2: shadow the variable (Go < 1.22 idiom) ────────────────────────
	// `i := i` creates a new variable in the inner scope that holds the
	// current value. The closure captures the inner i, not the loop variable.
	fmt.Println("  fix 2 — shadow the variable:")
	results2 := make(chan int, 5)
	for i := 0; i < 5; i++ {
		i := i // new variable per iteration
		wg.Add(1)
		go func() {
			defer wg.Done()
			results2 <- i
		}()
	}
	wg.Wait()
	close(results2)
	for v := range results2 {
		fmt.Printf("  %d", v)
	}
	fmt.Println()

	// ── Subtlety: closures that legitimately share a variable ────────────────
	// Not every closure capture is a bug. If you *want* goroutines to observe
	// updates to a shared variable, capturing by reference is correct —
	// just protect the access with a mutex or atomic.
	var counter int
	var mu sync.Mutex
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			mu.Lock()
			counter++ // intentional shared write
			mu.Unlock()
		}()
	}
	wg.Wait()
	fmt.Println("  shared counter:", counter)
}
