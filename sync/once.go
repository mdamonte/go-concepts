package main

import (
	"fmt"
	"sync"
)

// demoOnce shows sync.Once: a function passed to Do is executed exactly once,
// no matter how many goroutines call Do concurrently.
//
// Common uses:
//   - Lazy, thread-safe initialization of a singleton.
//   - One-time setup (DB connection, config load) shared across goroutines.
func demoOnce() {
	var once sync.Once
	var wg sync.WaitGroup

	init := func() {
		fmt.Println("  expensive init — runs exactly once")
	}

	// 10 goroutines all try to initialize at the same time.
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			once.Do(init) // only the first call executes init; the rest are no-ops
			fmt.Printf("  goroutine%d: init done\n", id)
		}(i)
	}

	wg.Wait()
}

// --- Singleton pattern using Once ---

type database struct{ dsn string }

var (
	dbInstance *database
	dbOnce     sync.Once
)

// getDB returns the single shared database connection, initializing it on
// the first call. Safe to call from multiple goroutines simultaneously.
func getDB() *database {
	dbOnce.Do(func() {
		fmt.Println("  [singleton] connecting to database...")
		dbInstance = &database{dsn: "postgres://localhost/mydb"}
	})
	return dbInstance
}
