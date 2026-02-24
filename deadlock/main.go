package main

import (
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"sync"
	"time"
)

var (
	muA sync.Mutex
	muB sync.Mutex
)

// goroutine1 locks A first, then tries to lock B.
func goroutine1(wg *sync.WaitGroup) {
	defer wg.Done()

	muA.Lock()
	fmt.Println("goroutine1: locked A")
	time.Sleep(50 * time.Millisecond) // let goroutine2 lock B first

	fmt.Println("goroutine1: waiting for B...") // will block here forever
	muB.Lock()
	defer muB.Unlock()
	defer muA.Unlock()

	fmt.Println("goroutine1: locked both (unreachable)")
}

// goroutine2 locks B first, then tries to lock A.
func goroutine2(wg *sync.WaitGroup) {
	defer wg.Done()

	muB.Lock()
	fmt.Println("goroutine2: locked B")
	time.Sleep(50 * time.Millisecond) // let goroutine1 lock A first

	fmt.Println("goroutine2: waiting for A...") // will block here forever
	muA.Lock()
	defer muA.Unlock()
	defer muB.Unlock()

	fmt.Println("goroutine2: locked both (unreachable)")
}

func main() {
	go func() {
		fmt.Println("pprof en http://localhost:6062/debug/pprof/")
		_ = http.ListenAndServe("localhost:6062", nil)
	}()

	var wg sync.WaitGroup
	wg.Add(2)

	go goroutine1(&wg)
	go goroutine2(&wg)

	wg.Wait() // blocks until both goroutines finish — they never will
}
