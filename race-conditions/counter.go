package main

import (
	"fmt"
	"sync"
	"sync/atomic"
)

const (
	goroutines = 100
	increments = 10_000
	expected   = goroutines * increments // 1_000_000
)

// demoCounterRace shows the most common race condition: two goroutines
// performing a read-modify-write on the same variable without synchronization.
//
// counter++ is NOT atomic. It compiles to three instructions:
//
//	LOAD  counter → reg
//	ADD   reg, 1
//	STORE reg → counter
//
// When two goroutines interleave between LOAD and STORE they both read the
// same value, both add 1, and both write back — one increment is lost.
//
// Run with -race to have the race detector flag every unsynchronized access:
//
//	go run -race .
func demoCounterRace() {
	var counter int
	var wg sync.WaitGroup

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < increments; j++ {
				counter++ // DATA RACE: read-modify-write is not atomic
			}
		}()
	}

	wg.Wait()
	fmt.Printf("  expected: %d  got: %d  lost updates: %d\n",
		expected, counter, expected-counter)
}

// demoCounterMutex fixes the race by wrapping the critical section with a Mutex.
// Only one goroutine can be inside Lock/Unlock at a time.
func demoCounterMutex() {
	var (
		counter int
		mu      sync.Mutex
		wg      sync.WaitGroup
	)

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < increments; j++ {
				mu.Lock()
				counter++ // protected: only one goroutine here at a time
				mu.Unlock()
			}
		}()
	}

	wg.Wait()
	fmt.Printf("  expected: %d  got: %d  ✓\n", expected, counter)
}

// demoCounterAtomic fixes the race with a single CPU instruction.
// Cheaper than a Mutex for simple numeric operations.
func demoCounterAtomic() {
	var counter atomic.Int64
	var wg sync.WaitGroup

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < increments; j++ {
				counter.Add(1) // atomic: single indivisible instruction
			}
		}()
	}

	wg.Wait()
	fmt.Printf("  expected: %d  got: %d  ✓\n", expected, counter.Load())
}

// demoCounterChannel fixes the race with the actor model: a single goroutine
// owns the counter and is the only one that reads or writes it.
// All other goroutines send increment requests via a channel.
//
// No shared memory → no race by construction.
func demoCounterChannel() {
	inc := make(chan struct{}, 512) // buffer absorbs bursts
	done := make(chan int)

	// Actor: sole owner of the counter.
	go func() {
		counter := 0
		for range inc {
			counter++
		}
		done <- counter
	}()

	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < increments; j++ {
				inc <- struct{}{}
			}
		}()
	}

	wg.Wait()
	close(inc)   // signal actor: no more increments
	counter := <-done
	fmt.Printf("  expected: %d  got: %d  ✓\n", expected, counter)
}
