package main

import (
	"fmt"
	"os"
	"sync"
	"time"
)

// demoSemacquire shows goroutine state [semacquire] / [sync.Mutex.Lock]:
// the goroutine is blocked waiting to acquire a mutex that another goroutine
// holds. This state appears for sync.Mutex, sync.RWMutex, sync.WaitGroup,
// sync.Cond, and channel operations backed by a runtime semaphore.
//
// Go < 1.22 shows the state as [semacquire].
// Go >= 1.22 shows the more descriptive label [sync.Mutex.Lock].
// The underlying function is always runtime_SemacquireMutex.
//
// Goroutine dump entry (Go 1.22+):
//
//	goroutine N [sync.Mutex.Lock]:
//	sync.runtime_SemacquireMutex(0x..., 0x0, 0x1)
//	sync.(*Mutex).lockSlow(...)
//	sync.(*Mutex).Lock(...)
//	main.demoSemacquire.func2()
//
// Cleanup: the holder releases the lock after the dump so the waiter
// can proceed — this is NOT a real deadlock, just the blocking state.
func demoSemacquire() {
	var mu sync.Mutex
	holderReady := make(chan struct{})
	releaseHolder := make(chan struct{})
	waiterDone := make(chan struct{})

	// Goroutine A: acquires the lock and holds it until signalled.
	go func() {
		mu.Lock()
		fmt.Println("  holder: acquired mu — will hold it")
		close(holderReady)
		<-releaseHolder
		mu.Unlock()
		fmt.Println("  holder: released mu")
	}()

	<-holderReady // wait until A holds the lock

	// Goroutine B: tries to acquire the same lock → blocks in [semacquire].
	go func() {
		defer close(waiterDone)
		fmt.Println("  waiter: blocking on mu.Lock() (held by holder)")
		mu.Lock() // ← blocked here, shows as [semacquire]
		fmt.Println("  waiter: acquired mu")
		mu.Unlock()
	}()

	time.Sleep(80 * time.Millisecond)
	dumpGoroutines()

	// Cleanup: release so the waiter can proceed.
	close(releaseHolder)
	<-waiterDone
}

// ── Classic AB deadlock ───────────────────────────────────────────────────────

var (
	muA sync.Mutex
	muB sync.Mutex
)

// goroutine1 locks A then waits for B.
func goroutine1(wg *sync.WaitGroup) {
	defer wg.Done()
	muA.Lock()
	fmt.Println("  goroutine1: locked A")
	time.Sleep(50 * time.Millisecond) // give goroutine2 time to lock B

	fmt.Println("  goroutine1: waiting for B...") // blocks here forever
	muB.Lock()
	defer muB.Unlock()
	defer muA.Unlock()
	fmt.Println("  goroutine1: locked both (unreachable)")
}

// goroutine2 locks B then waits for A.
func goroutine2(wg *sync.WaitGroup) {
	defer wg.Done()
	muB.Lock()
	fmt.Println("  goroutine2: locked B")
	time.Sleep(50 * time.Millisecond) // give goroutine1 time to lock A

	fmt.Println("  goroutine2: waiting for A...") // blocks here forever
	muA.Lock()
	defer muA.Unlock()
	defer muB.Unlock()
	fmt.Println("  goroutine2: locked both (unreachable)")
}

// demoMutexDeadlock shows the classic AB lock-ordering deadlock.
// After both goroutines are stuck it prints the goroutine dump — which now
// contains EVERY leaked goroutine from the earlier demos — and exits with
// code 1 to simulate a crash.
//
// Note: on macOS (and Linux after using net.Listen) the runtime's built-in
// deadlock detector is suppressed by the kqueue/epoll poller that stays
// active after the IO wait demo, so we trigger the dump manually.
// On a fresh program with no network code you would see:
//
//	fatal error: all goroutines are asleep - deadlock!
func demoMutexDeadlock() {
	var wg sync.WaitGroup
	wg.Add(2)
	go goroutine1(&wg)
	go goroutine2(&wg)

	// Wait long enough for both goroutines to reach their blocked states.
	time.Sleep(120 * time.Millisecond)

	fmt.Println("\n  ── goroutine dump (all states visible) ──")
	dumpGoroutines()

	fmt.Println("  ── simulated runtime panic ──")
	fmt.Println("  fatal error: all goroutines are asleep - deadlock!")
	fmt.Println()
	fmt.Println("  goroutine 1 [semacquire / sync.Mutex.Lock]:")
	fmt.Println("    → main goroutine (or wg.Wait) blocked on muA/muB")
	fmt.Println("  goroutine N [semacquire / sync.Mutex.Lock]:")
	fmt.Println("    → goroutine1 locked A, waiting for B")
	fmt.Println("  goroutine M [semacquire / sync.Mutex.Lock]:")
	fmt.Println("    → goroutine2 locked B, waiting for A")

	os.Exit(1) // non-zero exit simulates the runtime crash
}
