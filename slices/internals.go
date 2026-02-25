package main

import (
	"fmt"
	"unsafe"
)

// A slice is a 3-word struct (24 bytes on 64-bit):
//
//	+──────────+─────+─────+
//	│  ptr     │ len │ cap │   ← slice header, lives on the stack
//	+──────────+─────+─────+
//	     │
//	     ▼
//	[0][1][2][3][4][5]         ← backing array, lives on the heap
//
// ptr  → pointer to the first element visible through this slice
// len  → number of elements accessible via s[i]
// cap  → total elements from ptr to the end of the backing array
//
// Multiple slices can share the same backing array — this is the source of
// most slice-related bugs.

func demoInternals() {
	fmt.Printf("  sizeof([]int) = %d bytes (ptr+len+cap, 3×8 on 64-bit)\n",
		unsafe.Sizeof([]int{}))

	// ── Shared backing array ──────────────────────────────────────────────────
	fmt.Println("\n  Shared backing array:")
	a := []int{1, 2, 3, 4, 5}
	b := a[1:4] // b shares a's backing array; b = [2 3 4]

	printS("a", a)
	printS("b = a[1:4]", b)

	b[0] = 99 // writes to a[1] — both see the change
	fmt.Println("\n  after b[0] = 99:")
	printS("a", a) // [1 99 3 4 5]
	printS("b", b) // [99 3 4]

	// ── cap of a subslice ─────────────────────────────────────────────────────
	// cap(b) = cap(a) - 1 = 4, not len(b) = 3.
	// b can "see" a[4] via append (the next element in the backing array).
	fmt.Println("\n  cap of a subslice extends to end of backing array:")
	full := []int{10, 20, 30, 40, 50}
	sub := full[1:3]
	printS("full", full)
	printS("sub = full[1:3]", sub)
	// cap(sub) = 4 because the backing array still has full[3] and full[4] after sub.

	// ── Pass-by-value: the header is copied, not the array ───────────────────
	fmt.Println("\n  Pass-by-value — header is copied, backing array is shared:")
	s := []int{10, 20, 30}
	fmt.Printf("  before call: %v\n", s)

	modifyElement(s, 0, 99) // ✓ visible — writes to shared backing array
	fmt.Printf("  after modifyElement(s,0,99): %v\n", s)

	appendInside(s) // ✗ not visible — append modifies a copy of the header
	fmt.Printf("  after appendInside(s):       %v  ← unchanged\n", s)

	// ── append must be returned (or use a pointer to slice) ──────────────────
	fmt.Println("\n  To see append from a function, return the new slice:")
	s = appendAndReturn(s, 999)
	fmt.Printf("  after appendAndReturn: %v\n", s)
}

func modifyElement(s []int, i, v int) {
	s[i] = v // writes to the shared backing array — caller sees this
}

func appendInside(s []int) {
	s = append(s, 999) // s is a local copy of the header; caller's header unchanged
	fmt.Printf("  inside appendInside: %v\n", s)
}

func appendAndReturn(s []int, v int) []int {
	return append(s, v) // caller must reassign: s = appendAndReturn(s, v)
}

func printS(label string, s []int) {
	fmt.Printf("  %-18s %v  len=%d cap=%d\n", label+":", s, len(s), cap(s))
}
