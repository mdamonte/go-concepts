package main

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// Config simulates application configuration loaded at startup and reloaded
// on SIGHUP / periodic timer without restarting the process.
type Config struct {
	MaxConns int
	Timeout  time.Duration
	Feature  string
}

// demoValue shows atomic.Value: stores and loads an arbitrary value
// with atomic semantics.
//
// Rules:
//   - The concrete type stored must be the same on every Store call.
//   - The stored value must not be nil (use a pointer to a zero struct instead).
//   - Load returns nil if Store has never been called.
//
// Canonical use: hot-reload of read-mostly config. Many goroutines read
// at full speed; a single writer replaces the pointer atomically.
func demoValue() {
	var cfgVal atomic.Value

	// Initial configuration — stored as a pointer to allow nil-check on Load.
	cfgVal.Store(&Config{MaxConns: 10, Timeout: 5 * time.Second, Feature: "v1"})

	var wg sync.WaitGroup

	// Simulate 5 readers running concurrently with a writer.
	for i := range 5 {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			cfg := cfgVal.Load().(*Config) // always a consistent snapshot
			fmt.Printf("  reader %d: maxConns=%d feature=%s\n", id, cfg.MaxConns, cfg.Feature)
		}(i)
	}

	// Writer: replaces the config atomically.
	// Readers that already loaded the old pointer keep a valid copy;
	// new readers will see the updated config.
	wg.Add(1)
	go func() {
		defer wg.Done()
		time.Sleep(1 * time.Millisecond) // let readers start first
		cfgVal.Store(&Config{MaxConns: 50, Timeout: 10 * time.Second, Feature: "v2"})
		fmt.Println("  writer: config reloaded to v2")
	}()

	wg.Wait()

	// Confirm the final value.
	final := cfgVal.Load().(*Config)
	fmt.Printf("  final config: maxConns=%d feature=%s\n", final.MaxConns, final.Feature)
}
