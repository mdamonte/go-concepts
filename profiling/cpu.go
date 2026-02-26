package main

import (
	"fmt"
	"math/rand"
	"os"
	"runtime/pprof"
	"sort"
	"time"
)

// demoCPU writes a CPU profile to cpu.prof while running a CPU-bound workload.
//
// How it works:
//   The runtime samples the program counter of every running goroutine
//   at ~100 Hz (every 10ms). Each sample records the full call stack.
//   Functions that appear most often are the ones consuming CPU time.
//
// Inspect:
//
//	go tool pprof cpu.prof
//	(pprof) top            — top functions by cumulative CPU time
//	(pprof) top -flat      — top functions by self time (excluding callees)
//	(pprof) list sortWork  — annotated source for a specific function
//	(pprof) web            — flame graph in browser (requires graphviz)

func demoCPU() {
	f, err := os.Create("cpu.prof")
	if err != nil {
		fmt.Println("  error creating cpu.prof:", err)
		return
	}
	defer f.Close()

	// StartCPUProfile begins sampling. Only one CPU profile can run at a time.
	if err := pprof.StartCPUProfile(f); err != nil {
		fmt.Println("  error starting CPU profile:", err)
		return
	}

	fmt.Println("  CPU profile started — running workload...")
	start := time.Now()

	// Workload: sort and deduplicate random strings (realistic CPU work)
	result := 0
	for range 300 {
		result += sortWork(2000)
	}

	pprof.StopCPUProfile() // must call Stop before reading the file

	fmt.Printf("  done in %s  (result=%d)\n", time.Since(start).Round(time.Millisecond), result)
	fmt.Println("  profile written → cpu.prof")
	fmt.Println()
	fmt.Println("  Inspect:")
	fmt.Println("    go tool pprof cpu.prof")
	fmt.Println("    (pprof) top          — hottest functions")
	fmt.Println("    (pprof) top -flat    — self time only (no callees)")
	fmt.Println("    (pprof) list sortWork — annotated source")
	fmt.Println("    (pprof) web          — flame graph")
}

// sortWork is the CPU-bound function we want to see in the profile.
func sortWork(n int) int {
	data := make([]string, n)
	for i := range data {
		data[i] = fmt.Sprintf("item_%d", rand.Intn(n/2))
	}
	sort.Strings(data)

	if len(data) == 0 {
		return 0
	}
	deduped := data[:1]
	for _, s := range data[1:] {
		if s != deduped[len(deduped)-1] {
			deduped = append(deduped, s)
		}
	}
	return len(deduped)
}
