package main

import "fmt"

// ── Rule 1: LIFO — last deferred runs first ───────────────────────────────────
// defer is a stack: each defer pushes onto it; on return, the stack is popped.

func lifo() {
	fmt.Println("  body")
	defer fmt.Println("  defer 1  ← registered first, runs last")
	defer fmt.Println("  defer 2")
	defer fmt.Println("  defer 3  ← registered last, runs first")
}

// ── Rule 2: arguments are evaluated NOW, not when defer runs ─────────────────
// At the defer statement, all arguments are evaluated and stored.
// Changes to those variables after the defer statement have no effect.

func argEval() {
	x := 0
	defer fmt.Println("  defer fmt.Println(x): x was", x) // x=0 captured HERE
	x = 100
	fmt.Println("  at return, x =", x) // 100
	// defer prints 0 — the value when defer was registered
}

// ── Rule 3: closures capture the variable, not the value ─────────────────────
// A closure inside defer reads the variable when it executes,
// which is AFTER the function body completes.

func closureCapture() {
	x := 0
	defer func() { fmt.Println("  closure reads x:", x) }() // reads x when runs
	x = 100
	fmt.Println("  at return, x =", x) // 100
	// defer prints 100 — reads x after body finishes
}

// ── The classic trick question ────────────────────────────────────────────────
// "What does this print?"
// Answer: 2, 1, 0
//   - each defer captures i by VALUE (argument eval) → stores 0, 1, 2
//   - LIFO reverses them: 2, 1, 0

func trickArgEval() {
	fmt.Println("  defer fmt.Println(i) — arg eval, LIFO:")
	for i := range 3 {
		defer fmt.Printf("  i = %d\n", i) // i evaluated NOW: stores 0, then 1, then 2
	}
	// printed after this function returns, in reverse: 2, 1, 0
}

// ── Contrast: closure in loop ─────────────────────────────────────────────────
// With a closure, all defers share the same variable.
// In a classic C-style loop (Go ≤ 1.21 range semantics), they'd all print 3.
// In Go 1.22 range, each iteration has its own variable → still 2, 1, 0 here.
// The difference matters when the loop variable outlives the defer.

func trickClosure() {
	fmt.Println("  defer func(){...i...}() — closure, LIFO:")
	// Classic-style loop so the variable IS shared (same address every iteration)
	for i := 0; i < 3; i++ {
		i := i // shadow to get per-iteration variable (pre-Go 1.22 idiom)
		defer func() { fmt.Printf("  i = %d\n", i) }()
	}
	// prints 2, 1, 0 — because we shadowed i per iteration
}

func trickClosureGotcha() {
	fmt.Println("  closure WITHOUT shadow — all see same variable:")
	for i := 0; i < 3; i++ {
		defer func() { fmt.Printf("  i = %d\n", i) }() // captures &i, not a copy
	}
	// after loop, i = 3 → ALL defers print 3
}

func demoBasics() {
	fmt.Println("  ── LIFO ──")
	lifo()

	fmt.Println("\n  ── Argument evaluation at defer statement ──")
	argEval()

	fmt.Println("\n  ── Closure reads variable at execution time ──")
	closureCapture()

	fmt.Println("\n  ── Trick question 1: defer with arg in loop ──")
	trickArgEval()

	fmt.Println("\n  ── Trick question 2: closure in loop (shared variable) ──")
	trickClosureGotcha()

	fmt.Println("\n  ── Fix: shadow the variable per iteration ──")
	trickClosure()
}
