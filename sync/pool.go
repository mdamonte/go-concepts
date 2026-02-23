package main

import (
	"bytes"
	"fmt"
	"sync"
)

// demoPool shows sync.Pool: a cache of temporary objects that can be reused
// across goroutines to reduce heap allocations and GC pressure.
//
// Pool is ideal for short-lived, frequently allocated objects like
// bytes.Buffer, []byte, or encoder/decoder instances.
//
// Key properties:
//   - Get returns an existing object from the pool or calls New if empty.
//   - Put returns an object to the pool for future reuse.
//   - The GC may clear the pool at any time — do not store persistent state.
//   - Pool is safe for concurrent use without additional locking.
func demoPool() {
	pool := &sync.Pool{
		New: func() any {
			fmt.Println("  pool: allocating new buffer")
			return new(bytes.Buffer)
		},
	}

	// First Get: pool is empty → New is called.
	buf := pool.Get().(*bytes.Buffer)
	buf.WriteString("hello")
	fmt.Println("  got:", buf.String())

	// Reset before returning so the next caller gets a clean buffer.
	buf.Reset()
	pool.Put(buf) // return to pool

	// Second Get: reuses the buffer from the pool → New is NOT called.
	buf2 := pool.Get().(*bytes.Buffer)
	buf2.WriteString("world")
	fmt.Println("  got:", buf2.String())
	buf2.Reset()
	pool.Put(buf2)

	// Concurrent usage: each goroutine borrows and returns a buffer.
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			b := pool.Get().(*bytes.Buffer)
			defer func() {
				b.Reset()
				pool.Put(b)
			}()
			fmt.Fprintf(b, "goroutine%d", id)
			fmt.Println("  concurrent:", b.String())
		}(i)
	}
	wg.Wait()
}
