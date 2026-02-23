package main

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// demoPropagation shows the two key rules of context trees:
//
//  1. Cancelling a parent cancels ALL its descendants.
//  2. Cancelling a child does NOT affect its parent or siblings.
func demoPropagation() {
	parent, cancelParent := context.WithCancel(context.Background())
	defer cancelParent()

	// Two independent children derived from the same parent.
	child1, cancelChild1 := context.WithCancel(parent)
	defer cancelChild1()

	child2, cancelChild2 := context.WithTimeout(parent, 10*time.Second)
	defer cancelChild2()

	// Grandchild derived from child1.
	grandchild, cancelGrandchild := context.WithCancel(child1)
	defer cancelGrandchild()

	var wg sync.WaitGroup
	launch := func(name string, ctx context.Context) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-ctx.Done()
			fmt.Printf("  %-12s stopped → %v\n", name, ctx.Err())
		}()
	}

	launch("child1", child1)
	launch("child2", child2)
	launch("grandchild", grandchild)

	// ── Step 1: cancel child1 ──────────────────────────────────────────────
	// child1 and its grandchild stop. child2 and parent are unaffected.
	fmt.Println("cancelling child1...")
	cancelChild1()
	time.Sleep(30 * time.Millisecond)
	fmt.Printf("  parent alive: %v  child2 alive: %v\n",
		parent.Err() == nil, child2.Err() == nil)

	// ── Step 2: cancel parent ──────────────────────────────────────────────
	// All remaining descendants (child2) stop immediately.
	fmt.Println("cancelling parent...")
	cancelParent()

	wg.Wait()
}
