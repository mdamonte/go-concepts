package main

import (
	"fmt"
	"slices" // stdlib slices package — Go 1.21+
)

func demoOperations() {
	// ── copy ──────────────────────────────────────────────────────────────────
	// copy(dst, src) copies min(len(dst), len(src)) elements.
	// Returns the number of elements copied.
	// Safe with overlapping slices (same backing array).
	fmt.Println("  copy — min(len(dst), len(src)) elements:")
	src := []int{1, 2, 3, 4, 5}
	dst := make([]int, 3)
	n := copy(dst, src)
	fmt.Printf("  copy(dst[3], src[5]) = %d  dst=%v\n", n, dst)

	dst2 := make([]int, 7)
	n2 := copy(dst2, src)
	fmt.Printf("  copy(dst[7], src[5]) = %d  dst=%v\n", n2, dst2)

	// Shift left using copy on overlapping slices
	overlap := []int{1, 2, 3, 4, 5}
	copy(overlap[1:], overlap[2:]) // shift [2:] one position left
	overlap = overlap[:len(overlap)-1]
	fmt.Println("  shift-left via copy:", overlap) // [1 3 4 5]

	// ── Delete (order preserved) ──────────────────────────────────────────────
	// append(s[:i], s[i+1:]...) shifts everything left — O(n).
	fmt.Println("\n  Delete at index i (order preserved) — O(n):")
	d1 := []int{10, 20, 30, 40, 50}
	i := 2 // delete 30
	d1 = append(d1[:i], d1[i+1:]...)
	fmt.Println("  result:", d1) // [10 20 40 50]

	// ── Delete (order NOT preserved) ─────────────────────────────────────────
	// Swap with last element and shrink — O(1).
	fmt.Println("\n  Delete at index i (swap with last) — O(1), changes order:")
	d2 := []int{10, 20, 30, 40, 50}
	d2[i] = d2[len(d2)-1]
	d2 = d2[:len(d2)-1]
	fmt.Println("  result:", d2) // [10 20 50 40]

	// ── Insert ────────────────────────────────────────────────────────────────
	// Create a gap by appending and shifting, then fill it.
	fmt.Println("\n  Insert value at index i:")
	ins := []int{1, 2, 4, 5}
	i = 2
	ins = append(ins, 0)          // grow by 1 (may allocate)
	copy(ins[i+1:], ins[i:])      // shift right — copy handles overlap correctly
	ins[i] = 3
	fmt.Println("  result:", ins) // [1 2 3 4 5]

	// ── Filter in-place ───────────────────────────────────────────────────────
	// Reuse the backing array: write kept elements to the front.
	// No allocation — result shares memory with the original.
	// Do NOT use original after: it has stale data beyond result.
	fmt.Println("\n  Filter in-place (reuse backing array — zero allocation):")
	vals := []int{1, 2, 3, 4, 5, 6, 7, 8}
	result := vals[:0] // same backing array, len=0
	for _, v := range vals {
		if v%2 == 0 {
			result = append(result, v) // writes into vals' backing array
		}
	}
	fmt.Println("  evens:", result)

	// ── Reverse in-place ─────────────────────────────────────────────────────
	fmt.Println("\n  Reverse in place:")
	rev := []int{1, 2, 3, 4, 5}
	for lo, hi := 0, len(rev)-1; lo < hi; lo, hi = lo+1, hi-1 {
		rev[lo], rev[hi] = rev[hi], rev[lo]
	}
	fmt.Println("  reversed:", rev)

	// ── Deduplicate (sorted slice) ────────────────────────────────────────────
	fmt.Println("\n  Deduplicate a sorted slice:")
	sorted := []int{1, 1, 2, 3, 3, 3, 4, 5, 5}
	deduped := sorted[:1]
	for _, v := range sorted[1:] {
		if v != deduped[len(deduped)-1] {
			deduped = append(deduped, v)
		}
	}
	fmt.Println("  deduped:", deduped)

	// ── stdlib slices package (Go 1.21+) ─────────────────────────────────────
	// For production code, prefer the stdlib over manual implementations.
	fmt.Println("\n  stdlib slices package (Go 1.21+):")
	s := []int{5, 3, 1, 4, 2}
	slices.Sort(s)
	fmt.Println("  slices.Sort:", s)
	fmt.Println("  slices.Contains([1..5], 3):", slices.Contains(s, 3))
	fmt.Println("  slices.Index([1..5], 4):   ", slices.Index(s, 4))

	s2 := []int{10, 20, 30, 40, 50}
	s2 = slices.Delete(s2, 1, 3) // delete s2[1:3]
	fmt.Println("  slices.Delete([10..50], 1, 3):", s2)

	s3 := []int{1, 1, 2, 3, 3}
	fmt.Println("  slices.Compact (dedup sorted):", slices.Compact(s3))
}
