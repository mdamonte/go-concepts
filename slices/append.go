package main

import "fmt"

// append has two distinct behaviors depending on capacity:
//
//   len < cap  → writes element into the existing backing array (no allocation)
//   len == cap → allocates a new, larger array, copies all elements, then appends
//
// Growth factor (Go 1.18+): roughly 2× for small slices, tapering toward 1.25×
// for slices above ~256 elements. The exact formula lives in runtime/slice.go
// and can change between Go versions — never rely on a specific growth factor.

func demoAppend() {
	// ── Growth visualization ──────────────────────────────────────────────────
	fmt.Println("  Growth — capacity doubles until ~256 elements, then grows ~1.25×:")
	s := make([]int, 0)
	prevCap := 0
	for i := range 18 {
		s = append(s, i)
		if cap(s) != prevCap {
			fmt.Printf("  len=%-3d  cap grew %d → %d\n", len(s), prevCap, cap(s))
			prevCap = cap(s)
		}
	}

	// ── In-place append (len < cap) ───────────────────────────────────────────
	fmt.Println("\n  In-place append (len < cap — no allocation):")
	pre := make([]int, 3, 6) // len=3, cap=6 — room for 3 more
	printS("before", pre)
	pre = append(pre, 99)
	printS("after append(99)", pre) // cap unchanged

	// ── THE classic gotcha: append to a subslice ──────────────────────────────
	// When cap(sub) > len(sub), appending to sub writes into the original
	// backing array — silently overwriting elements the caller still holds.
	fmt.Println("\n  Gotcha — append to subslice overwrites original when cap allows:")
	orig := []int{1, 2, 3, 4, 5}
	sub := orig[1:3] // sub = [2 3]; cap(sub) = 4 (orig[3] and orig[4] are still reachable)
	printS("orig", orig)
	printS("sub = orig[1:3]", sub)

	sub = append(sub, 99) // cap=4 > len=2 → writes 99 into orig[3]!
	fmt.Println("\n  after sub = append(sub, 99):")
	printS("orig", orig) // orig[3] silently changed to 99
	printS("sub", sub)

	// ── Fix: 3-index slice s[low:high:max] ───────────────────────────────────
	// The third index sets the cap of the resulting slice.
	// cap(s[low:high:max]) = max - low
	//
	// Use it to prevent append from reaching into the original array.
	fmt.Println("\n  Fix — 3-index slice s[low:high:max] caps the subslice capacity:")
	orig2 := []int{1, 2, 3, 4, 5}
	safe := orig2[1:3:3] // cap = 3-1 = 2, same as len → append must allocate
	printS("orig2", orig2)
	printS("safe = orig2[1:3:3]", safe)

	safe = append(safe, 99) // cap exhausted → new backing array allocated
	fmt.Println("\n  after safe = append(safe, 99):")
	printS("orig2", orig2) // untouched
	printS("safe", safe)   // new backing array

	// ── Pre-allocate for known sizes ──────────────────────────────────────────
	// Without pre-allocation, each capacity overflow copies all elements.
	// With make([]T, 0, n), a single allocation handles everything.
	fmt.Println("\n  Efficiency — pre-allocate with make([]T, 0, n):")
	result := make([]int, 0, 10)
	for i := range 10 {
		result = append(result, i*i)
	}
	fmt.Println("  result:", result)
	fmt.Printf("  len=%d cap=%d (no reallocation occurred)\n", len(result), cap(result))
}
