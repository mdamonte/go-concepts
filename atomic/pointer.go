package main

import (
	"fmt"
	"sync"
	"sync/atomic"
)

// Node is a simple immutable snapshot used to demo atomic.Pointer.
type Node struct {
	Value int
	Label string
}

// demoPointer shows atomic.Pointer[T] (Go 1.19+): a type-safe atomic
// pointer to any type T.
//
// Advantages over atomic.Value:
//   - Nil is a valid stored value (unlike atomic.Value).
//   - No interface allocation — T is inlined in the generic instantiation.
//   - CompareAndSwap works on the pointer itself.
//
// Use case: publishing an immutable snapshot that many goroutines read
// while one writer replaces the whole struct atomically.
func demoPointer() {
	var ptr atomic.Pointer[Node]

	// ptr.Load() returns nil before any Store.
	fmt.Println("  before Store:", ptr.Load()) // <nil>

	ptr.Store(&Node{Value: 1, Label: "initial"})
	n := ptr.Load()
	fmt.Printf("  after Store:  value=%d label=%s\n", n.Value, n.Label)

	// Swap: replace and get the old pointer.
	old := ptr.Swap(&Node{Value: 2, Label: "updated"})
	fmt.Printf("  Swap: old={%d %s} new={%d %s}\n",
		old.Value, old.Label,
		ptr.Load().Value, ptr.Load().Label,
	)

	// CompareAndSwap on the pointer.
	current := ptr.Load()
	swapped := ptr.CompareAndSwap(current, &Node{Value: 3, Label: "cas"})
	fmt.Printf("  CAS: swapped=%v val=%d label=%s\n", swapped, ptr.Load().Value, ptr.Load().Label)

	// Concurrent publish pattern: one writer, many readers.
	var latest atomic.Pointer[Node]
	latest.Store(&Node{Value: 0, Label: "start"})

	var wg sync.WaitGroup
	for i := 1; i <= 3; i++ {
		wg.Add(1)
		go func(v int) {
			defer wg.Done()
			// Replace the shared pointer with a fresh immutable struct.
			latest.Store(&Node{Value: v, Label: fmt.Sprintf("gen-%d", v)})
		}(i)
	}
	wg.Wait()

	snap := latest.Load()
	fmt.Printf("  final snapshot: value=%d label=%s\n", snap.Value, snap.Label)
}
