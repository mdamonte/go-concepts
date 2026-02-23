package main

import (
	"fmt"
	"sync"
)

// demoBasics shows the three ways to launch a goroutine and the key
// difference between calling a function and launching it as a goroutine.
func demoBasics() {
	var wg sync.WaitGroup

	// 1. Named function — clearest form; gives the goroutine a readable stack trace.
	wg.Add(1)
	go greet("Alice", &wg)

	// 2. Anonymous function — inline, useful for short closures.
	wg.Add(1)
	go func() {
		defer wg.Done()
		fmt.Println("  anonymous goroutine running")
	}()

	// 3. Method on a value — goroutines can call methods too.
	wg.Add(1)
	w := worker{id: 7}
	go w.run(&wg)

	// Main continues immediately after each `go` statement.
	// Without wg.Wait() the program might exit before the goroutines run.
	fmt.Println("  main: all goroutines launched")
	wg.Wait()
	fmt.Println("  main: all goroutines done")
}

func greet(name string, wg *sync.WaitGroup) {
	defer wg.Done()
	fmt.Printf("  hello from goroutine, %s\n", name)
}

type worker struct{ id int }

func (w worker) run(wg *sync.WaitGroup) {
	defer wg.Done()
	fmt.Printf("  worker%d running\n", w.id)
}
