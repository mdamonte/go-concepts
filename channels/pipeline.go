package main

import (
	"fmt"
	"sync"
)

// demoPipeline shows the pipeline pattern: a series of stages connected
// by channels, each consuming values from the previous stage and producing
// values for the next.
//
//	generate → square → print
//
// Each stage is a goroutine; channels act as the conveyor belt between them.
// The pipeline is lazy: each stage only runs when the next stage pulls.
func demoPipeline() {
	// Stage 1: emit integers.
	naturals := generate(2, 3, 4, 5, 6, 7, 8, 9)

	// Stage 2: keep only primes.
	primes := filterPrimes(naturals)

	// Stage 3: square the primes.
	squared := square(primes)

	// Consume and print; range exits when squared is closed.
	for v := range squared {
		fmt.Printf("%d ", v)
	}
	fmt.Println()
}

// generate sends each value to a new channel and closes it when done.
// Returns a receive-only channel — callers can only read from it.
func generate(nums ...int) <-chan int {
	out := make(chan int)
	go func() {
		defer close(out)
		for _, n := range nums {
			out <- n
		}
	}()
	return out
}

// filterPrimes passes only prime numbers downstream.
func filterPrimes(in <-chan int) <-chan int {
	out := make(chan int)
	go func() {
		defer close(out)
		for n := range in {
			if isPrime(n) {
				out <- n
			}
		}
	}()
	return out
}

// square multiplies each value by itself.
func square(in <-chan int) <-chan int {
	out := make(chan int)
	go func() {
		defer close(out)
		for n := range in {
			out <- n * n
		}
	}()
	return out
}

func isPrime(n int) bool {
	if n < 2 {
		return false
	}
	for i := 2; i*i <= n; i++ {
		if n%i == 0 {
			return false
		}
	}
	return true
}

// ── Fan-out ───────────────────────────────────────────────────────────────────

// demoFanOut distributes work from a single input channel across multiple
// goroutines, processing items in parallel.
// Useful when each unit of work is independent and CPU-bound.
func demoFanOut() {
	jobs := generate(1, 2, 3, 4, 5, 6, 7, 8)

	// Start 3 workers, each reading from the same jobs channel.
	const numWorkers = 3
	results := make([]<-chan int, numWorkers)
	for i := 0; i < numWorkers; i++ {
		results[i] = squareWorker(i, jobs)
	}

	// Merge and print all results.
	for v := range merge(results...) {
		fmt.Printf("%d ", v)
	}
	fmt.Println()
}

func squareWorker(id int, in <-chan int) <-chan int {
	out := make(chan int)
	go func() {
		defer close(out)
		for n := range in {
			fmt.Printf("  worker%d: %d²\n", id, n)
			out <- n * n
		}
	}()
	return out
}

// ── Fan-in (merge) ────────────────────────────────────────────────────────────

// demoFanIn shows how to merge multiple channels into a single channel.
// Each goroutine forwards values from one input channel to the merged output.
func demoFanIn() {
	a := generate(10, 20, 30)
	b := generate(100, 200, 300)
	c := generate(1000, 2000, 3000)

	for v := range merge(a, b, c) {
		fmt.Printf("%d ", v)
	}
	fmt.Println()
}

// merge fans-in any number of input channels into a single output channel.
// It closes the output when all inputs are exhausted.
func merge(cs ...<-chan int) <-chan int {
	out := make(chan int)
	var wg sync.WaitGroup

	forward := func(ch <-chan int) {
		defer wg.Done()
		for v := range ch {
			out <- v
		}
	}

	wg.Add(len(cs))
	for _, ch := range cs {
		go forward(ch)
	}

	// Close out once all inputs are done.
	go func() {
		wg.Wait()
		close(out)
	}()

	return out
}
