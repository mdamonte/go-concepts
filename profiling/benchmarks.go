package main

import (
	"fmt"
	"strings"
	"testing"
)

// demoBenchmarks uses testing.AllocsPerRun to show allocation differences
// without needing the test runner. For full benchmarks see bench_test.go.
//
// Run actual benchmarks:
//
//	go test -bench=. -benchmem
//	go test -bench=BenchmarkStringBuilder -count=5
//	go test -bench=. -benchtime=3s -cpuprofile=bench_cpu.prof

func demoBenchmarks() {
	fmt.Println("  testing.AllocsPerRun — measure allocations outside test runner:")
	fmt.Println()

	// string += vs strings.Builder vs Builder+Grow
	alloc := func(label string, n int, f func()) {
		a := testing.AllocsPerRun(50, f)
		fmt.Printf("  %-32s  %.0f allocs\n", label, a)
	}

	alloc("string += (200 iters)", 200, func() {
		s := ""
		for range 200 {
			s += "x"
		}
		sink = s
	})

	alloc("strings.Builder (200 iters)", 200, func() {
		var b strings.Builder
		for range 200 {
			b.WriteByte('x')
		}
		sink = b.String()
	})

	alloc("Builder + Grow(200)", 200, func() {
		var b strings.Builder
		b.Grow(200)
		for range 200 {
			b.WriteByte('x')
		}
		sink = b.String()
	})

	fmt.Println()

	alloc("append (no cap, 1000 ints)", 1000, func() {
		var s []int
		for i := range 1000 {
			s = append(s, i)
		}
		sink = s
	})

	alloc("make([]int, 0, 1000)", 1000, func() {
		s := make([]int, 0, 1000)
		for i := range 1000 {
			s = append(s, i)
		}
		sink = s
	})

	fmt.Println()
	fmt.Println("  Run bench_test.go for full benchmarks with ns/op and B/op:")
	fmt.Println("    go test -bench=. -benchmem")
	fmt.Println("    go test -bench=. -count=5 -benchtime=3s")
	fmt.Println("    go test -bench=. -cpuprofile=bench_cpu.prof -memprofile=bench_mem.prof")
	fmt.Println()
	fmt.Println("  Compare two versions with benchstat:")
	fmt.Println("    go test -bench=. -count=10 > old.txt")
	fmt.Println("    # make changes")
	fmt.Println("    go test -bench=. -count=10 > new.txt")
	fmt.Println("    benchstat old.txt new.txt")
}
