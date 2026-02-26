package main

import "fmt"

// defer in a loop — the most common resource-leak bug in Go.
//
// defer does NOT run at the end of each loop iteration.
// It runs when the ENCLOSING FUNCTION returns.
// In a loop that opens N resources, all N defers fire at function exit —
// meaning all resources stay open for the entire duration of the function.

// resource simulates an I/O resource (file, DB connection, HTTP body...).
type resource struct {
	id     int
	closed bool
}

var opened, closed int // counters to observe the leak

func openRes(id int) *resource {
	opened++
	fmt.Printf("    open(%d)\n", id)
	return &resource{id: id}
}

func (r *resource) Close() {
	if r.closed {
		return
	}
	r.closed = true
	closed++
	fmt.Printf("    close(%d)\n", r.id)
}

// ── WRONG: all closes happen when processWrong returns ───────────────────────
// If n is large, you hold n file descriptors simultaneously.
// If the function is long-running, resources leak for its entire duration.
func processWrong(n int) {
	for i := range n {
		r := openRes(i)
		defer r.Close() // ← deferred until processWrong returns, not loop iteration
		// ... use r ...
		fmt.Printf("    processing %d (resource still open)\n", i)
	}
	fmt.Println("    all iterations done — defers fire NOW:")
}

// ── Fix 1: extract to a helper function ──────────────────────────────────────
// The helper's defer fires when the helper returns (end of each iteration).
// This is the clearest fix and the most idiomatic.
func processFixed1(n int) {
	for i := range n {
		processOne(i)
	}
}

func processOne(id int) {
	r := openRes(id)
	defer r.Close() // fires when processOne returns → end of each iteration
	fmt.Printf("    processing %d\n", id)
}

// ── Fix 2: immediately-invoked function literal ───────────────────────────────
// Same effect as fix 1 without a named helper.
// Useful for short one-off logic; can feel noisy for complex logic.
func processFixed2(n int) {
	for i := range n {
		func() {
			r := openRes(i)
			defer r.Close() // fires when this closure returns → end of each iteration
			fmt.Printf("    processing %d\n", i)
		}()
	}
}

// ── Fix 3: explicit close — no defer ─────────────────────────────────────────
// Simplest when error handling is straightforward.
// Risk: early returns or panics skip the close.
// Use only when the body is short and has no early returns.
func processFixed3(n int) {
	for i := range n {
		r := openRes(i)
		fmt.Printf("    processing %d\n", i)
		r.Close() // explicit — no defer, must be called on every exit path
	}
}

func demoLoops() {
	fmt.Println("  WRONG — defer in loop: all closes happen at function return:")
	opened, closed = 0, 0
	processWrong(3)
	fmt.Printf("  opened=%d closed=%d at function return\n", opened, closed)

	fmt.Println("\n  Fix 1 — helper function: close at end of each iteration:")
	opened, closed = 0, 0
	processFixed1(3)
	fmt.Printf("  opened=%d closed=%d (resources closed promptly)\n", opened, closed)

	fmt.Println("\n  Fix 2 — immediately-invoked closure:")
	opened, closed = 0, 0
	processFixed2(3)
	fmt.Printf("  opened=%d closed=%d\n", opened, closed)

	fmt.Println("\n  Fix 3 — explicit close (no defer):")
	opened, closed = 0, 0
	processFixed3(3)
	fmt.Printf("  opened=%d closed=%d\n", opened, closed)
}
