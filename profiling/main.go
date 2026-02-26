package main

import "fmt"

// Profiling in Go — covers all types that appear in technical interviews.
//
// Run:
//
//	go run .                       — generates cpu.prof, mem.prof, goroutine.prof
//	go test -bench=. -benchmem     — run benchmarks (see bench_test.go)
func main() {
	section("CPU profiling — pprof.StartCPUProfile / StopCPUProfile")
	demoCPU()

	section("Memory profiling — pprof.WriteHeapProfile, allocation comparison")
	demoMemory()

	section("Named profiles — goroutine, block, mutex via pprof.Lookup")
	demoNamedProfiles()

	section("HTTP pprof — net/http/pprof endpoints for production services")
	demoHTTPPprof()

	section("Benchmarks — testing.AllocsPerRun (see bench_test.go for testing.B)")
	demoBenchmarks()
}

func section(title string) {
	fmt.Printf("\n━━━ %s ━━━\n", title)
}
