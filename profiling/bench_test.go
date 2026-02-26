package main

import (
	"fmt"
	"strings"
	"testing"
)

// Run:
//
//	go test -bench=. -benchmem
//	go test -bench=BenchmarkString -benchmem -count=5
//	go test -bench=. -cpuprofile=bench_cpu.prof
//	go test -bench=. -memprofile=bench_mem.prof

// ── b.N — the benchmark loop ──────────────────────────────────────────────────
// The test runner adjusts b.N until the benchmark runs for at least 1 second.
// Never set b.N yourself; only iterate over it.

func BenchmarkStringConcat(b *testing.B) {
	for range b.N {
		s := ""
		for range 100 {
			s += "x" // O(n²) allocations
		}
		sink = s
	}
}

func BenchmarkStringBuilder(b *testing.B) {
	for range b.N {
		var sb strings.Builder
		for range 100 {
			sb.WriteByte('x')
		}
		sink = sb.String()
	}
}

func BenchmarkStringBuilderPrealloc(b *testing.B) {
	for range b.N {
		var sb strings.Builder
		sb.Grow(100) // single allocation
		for range 100 {
			sb.WriteByte('x')
		}
		sink = sb.String()
	}
}

// ── b.ResetTimer — exclude setup from measurement ────────────────────────────

func BenchmarkWithSetup(b *testing.B) {
	// Expensive setup — must NOT count toward ns/op
	data := make([]int, 100_000)
	for i := range data {
		data[i] = i
	}

	b.ResetTimer() // ← reset here; everything above is excluded

	for range b.N {
		sum := 0
		for _, v := range data {
			sum += v
		}
		sink = sum
	}
}

// ── b.StopTimer / b.StartTimer — pause around per-iteration setup ─────────────

func BenchmarkWithPerIterSetup(b *testing.B) {
	for range b.N {
		b.StopTimer()
		data := make([]int, 1000) // per-iteration setup, not measured
		for i := range data {
			data[i] = 1000 - i
		}
		b.StartTimer()

		// Only this part is measured:
		sum := 0
		for _, v := range data {
			sum += v
		}
		sink = sum
	}
}

// ── b.ReportAllocs — explicit alloc reporting ─────────────────────────────────
// Equivalent to passing -benchmem for this benchmark only.

func BenchmarkSprintf(b *testing.B) {
	b.ReportAllocs()
	for range b.N {
		sink = fmt.Sprintf("key=%d value=%s", 42, "hello")
	}
}

// ── b.RunParallel — concurrent load ──────────────────────────────────────────
// Each goroutine gets its own *testing.PB and calls pb.Next() to iterate.
// Use to benchmark goroutine-safe code under concurrent load.

func BenchmarkParallel(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			sink = fmt.Sprintf("item_%d", 42)
		}
	})
}

// ── Sub-benchmarks — compare multiple implementations ────────────────────────
// go test -bench=BenchmarkSlice/.*

func BenchmarkSlice(b *testing.B) {
	b.Run("append_no_cap", func(b *testing.B) {
		for range b.N {
			var s []int
			for i := range 1000 {
				s = append(s, i)
			}
			sink = s
		}
	})

	b.Run("make_with_cap", func(b *testing.B) {
		for range b.N {
			s := make([]int, 0, 1000)
			for i := range 1000 {
				s = append(s, i)
			}
			sink = s
		}
	})

	b.Run("make_with_len", func(b *testing.B) {
		for range b.N {
			s := make([]int, 1000)
			for i := range 1000 {
				s[i] = i
			}
			sink = s
		}
	})
}
