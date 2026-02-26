package main

import (
	"fmt"
	"runtime"
)

// defer and panic — four rules to know for interviews.
//
//  1. defer runs during panic unwinding — before the goroutine dies.
//  2. recover() stops a panic ONLY when called directly inside a deferred func.
//  3. After recover(), execution continues after the deferred func — NOT after panic().
//  4. os.Exit() bypasses defer entirely — defers DO NOT run.

// ── Rule 1: defer runs during panic ──────────────────────────────────────────

func deferRunsDuringPanic() (result string) {
	defer func() {
		result = "defer ran" // proof that defer executed
		fmt.Println("  defer: I ran during panic unwind")
	}()
	panic("oh no")
}

// ── Rule 2: recover() stops panic ONLY from a direct defer ───────────────────

func recoversCorrectly() {
	defer func() {
		if r := recover(); r != nil { // ✓ called directly inside defer
			fmt.Println("  recovered:", r)
		}
	}()
	panic("caught!")
}

// This does NOT work — recover() inside a helper called from defer returns nil.
func recoverFromHelper() {
	defer recoverHelper() // ✗ recover() is not in a DIRECT deferred function
	panic("this will NOT be caught")
}

func recoverHelper() {
	if r := recover(); r != nil { // returns nil — not a direct defer context
		fmt.Println("  helper recovered:", r)
	}
	// panic continues propagating
}

// ── Rule 3: execution resumes after the deferred func, not after panic() ─────

func resumeAfterRecover() (result string) {
	result = "initial"
	defer func() {
		if r := recover(); r != nil {
			result = "recovered" // modifies the named return variable
			fmt.Println("  inside recover: result =", result)
			// execution does NOT jump back to where panic() was called
			// it continues here, finishes this defer, then function returns
		}
	}()

	fmt.Println("  before panic")
	panic("oops")
}

// ── Rule 4: os.Exit bypasses defer ───────────────────────────────────────────
// os.Exit terminates the process immediately — defers do NOT run.
// log.Fatal, log.Fatalf, log.Fatalln all call os.Exit(1) internally.
//
// This means:
//   defer file.Close()
//   log.Fatal("error")   ← file is NOT closed
//
// Never use log.Fatal if you have important cleanup in defers.
// Use log.Print + return/os.Exit at the top level instead.

// ── Practical: re-panic for unexpected panics ─────────────────────────────────
// Catch only the panics you expect; let runtime errors (nil pointer,
// index out of bounds) propagate so bugs aren't silently swallowed.
//
// Note: integer division by zero (a/b when b==0) is a runtime.Error —
// so safeDiv panics explicitly with a string value to demonstrate recovery.
// The runtime.Error branch covers nil dereferences, out-of-bounds, etc.

func safeDiv(a, b int) (result int, err error) {
	defer func() {
		if r := recover(); r != nil {
			switch e := r.(type) {
			case runtime.Error:
				// runtime panics (nil deref, out-of-bounds, etc.) — re-panic
				// these are programming errors, not expected conditions
				panic(e)
			default:
				err = fmt.Errorf("recovered panic: %v", r)
			}
		}
	}()
	if b == 0 {
		panic("division by zero") // explicit string panic — NOT a runtime.Error
	}
	return a / b, nil
}

// ── safeGo: run a goroutine with panic recovery ───────────────────────────────
// A goroutine that panics crashes the whole program — there is no way to
// recover from another goroutine. Each goroutine must recover itself.

func safeGo(fn func()) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("  safeGo recovered: %v\n", r)
			}
		}()
		fn()
	}()
}

func demoPanic() {
	fmt.Println("  Rule 1: defer runs during panic unwind:")
	func() {
		defer func() { recover() }() // prevent program crash
		result := deferRunsDuringPanic()
		fmt.Println("  result after recover:", result)
	}()

	fmt.Println("\n  Rule 2a: recover() called directly in defer — works:")
	recoversCorrectly()

	fmt.Println("\n  Rule 2b: recover() called inside a helper — does NOT work:")
	func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Println("  outer recovered what helper missed:", r)
			}
		}()
		recoverFromHelper()
	}()

	fmt.Println("\n  Rule 3: execution resumes after deferred func, not after panic:")
	result := resumeAfterRecover()
	fmt.Println("  function returned:", result)

	fmt.Println("\n  Rule 4: os.Exit bypasses defer (shown as comment — would exit):")
	fmt.Println("  defer file.Close()")
	fmt.Println("  log.Fatal(err)     ← calls os.Exit(1) — defer DOES NOT run")
	fmt.Println("  // Use log.Print + return instead when cleanup defers exist")

	fmt.Println("\n  safeDiv — re-panic on runtime.Error, recover on expected panics:")
	res, err := safeDiv(10, 2)
	fmt.Printf("  safeDiv(10, 2) → %d %v\n", res, err)
	res, err = safeDiv(10, 0)
	fmt.Printf("  safeDiv(10, 0) → %d %v (division by zero recovered)\n", res, err)

	fmt.Println("\n  safeGo — goroutine with built-in recovery:")
	done := make(chan struct{})
	safeGo(func() {
		defer close(done)
		panic("goroutine panic — caught by safeGo")
	})
	<-done
}
