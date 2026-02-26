package main

import (
	"fmt"
	"os"
	"runtime"
	"runtime/pprof"
	"strings"
)

// demoMemory shows allocation cost differences and writes a heap profile.
//
// Heap profile types:
//
//	inuse_space   — bytes currently in use (after GC) — find memory leaks
//	inuse_objects — count of live objects
//	alloc_space   — total bytes ever allocated (including freed) — find GC pressure
//	alloc_objects — total allocation count (including freed)
//
// Inspect:
//
//	go tool pprof mem.prof
//	(pprof) top -inuse_space    — what's alive right now
//	(pprof) top -alloc_space    — what caused the most GC pressure
//	(pprof) top -alloc_objects  — what allocated the most objects

// sink prevents the compiler from optimising away heap allocations in demos.
var sink interface{}

func demoMemory() {
	// ── Allocation comparison ─────────────────────────────────────────────────
	fmt.Println("  Allocation comparison (measured with runtime.ReadMemStats):")

	// string += : O(n²) — each concatenation allocates a new string
	alloc1 := measureAlloc(func() {
		s := ""
		for range 200 {
			s += "x"
		}
		sink = s
	})

	// strings.Builder : O(n) — one backing buffer, doubled as needed
	alloc2 := measureAlloc(func() {
		var b strings.Builder
		for range 200 {
			b.WriteByte('x')
		}
		sink = b.String()
	})

	// strings.Builder with Grow : O(1) — single allocation upfront
	alloc3 := measureAlloc(func() {
		var b strings.Builder
		b.Grow(200)
		for range 200 {
			b.WriteByte('x')
		}
		sink = b.String()
	})

	fmt.Printf("  string +=        200 iters → %6d bytes allocated\n", alloc1)
	fmt.Printf("  strings.Builder  200 iters → %6d bytes allocated\n", alloc2)
	fmt.Printf("  Builder + Grow   200 iters → %6d bytes allocated\n", alloc3)

	// ── Slice pre-allocation ──────────────────────────────────────────────────
	fmt.Println()
	alloc4 := measureAlloc(func() {
		var s []int
		for i := range 1000 {
			s = append(s, i) // multiple reallocations
		}
		sink = s
	})
	alloc5 := measureAlloc(func() {
		s := make([]int, 0, 1000) // single allocation
		for i := range 1000 {
			s = append(s, i)
		}
		sink = s
	})
	fmt.Printf("  append (no cap)  1000 ints → %6d bytes allocated\n", alloc4)
	fmt.Printf("  make([]int,0,n)  1000 ints → %6d bytes allocated\n", alloc5)

	// ── Write heap profile ────────────────────────────────────────────────────
	// Allocate objects so the profile has interesting content.
	holdLargeObjects()

	f, err := os.Create("mem.prof")
	if err != nil {
		fmt.Println("\n  error creating mem.prof:", err)
		return
	}
	defer f.Close()

	// GC before writing so inuse_space shows only live objects.
	runtime.GC()

	if err := pprof.WriteHeapProfile(f); err != nil {
		fmt.Println("\n  error writing heap profile:", err)
		return
	}

	fmt.Println("\n  heap profile written → mem.prof")
	fmt.Println()
	fmt.Println("  Inspect:")
	fmt.Println("    go tool pprof mem.prof")
	fmt.Println("    (pprof) top -inuse_space    — live objects (memory leaks)")
	fmt.Println("    (pprof) top -alloc_space    — GC pressure (total allocated)")
	fmt.Println("    (pprof) top -alloc_objects  — allocation count")
}

// measureAlloc returns the net bytes allocated by f (excluding GC collections).
func measureAlloc(f func()) uint64 {
	runtime.GC()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	f()
	runtime.GC()
	runtime.ReadMemStats(&after)
	if after.TotalAlloc < before.TotalAlloc {
		return 0
	}
	return after.TotalAlloc - before.TotalAlloc
}

// holdLargeObjects keeps objects alive so they appear in inuse_space.
func holdLargeObjects() {
	bufs := make([][]byte, 50)
	for i := range bufs {
		bufs[i] = make([]byte, 4*1024) // 4 KB each → 200 KB total
	}
	sink = bufs // prevent GC
}
