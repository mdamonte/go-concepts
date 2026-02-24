package main

import (
	"errors"
	"fmt"
)

// Sentinel errors are package-level variables that represent a fixed,
// well-known error condition. Callers compare against them with errors.Is.
//
// Convention: name them Err<Condition>. Keep the message lowercase and
// without punctuation (Go style).
var (
	ErrNotFound   = errors.New("not found")
	ErrPermission = errors.New("permission denied")
	ErrTimeout    = errors.New("operation timed out")
)

// findUser simulates a repository lookup that may return sentinel errors.
func findUser(id int) (string, error) {
	switch id {
	case 1:
		return "alice", nil
	case 2:
		return "", ErrPermission
	default:
		return "", fmt.Errorf("findUser %d: %w", id, ErrNotFound)
	}
}

// demoSentinel shows how to define sentinel errors and use errors.Is to
// detect them even when wrapped inside another error.
//
// errors.Is walks the entire Unwrap chain — it is NOT a simple == comparison.
// This means you can wrap a sentinel with context and still detect it.
func demoSentinel() {
	ids := []int{1, 2, 99}

	for _, id := range ids {
		name, err := findUser(id)
		if err == nil {
			fmt.Printf("  id=%d → user=%q\n", id, name)
			continue
		}

		switch {
		case errors.Is(err, ErrNotFound):
			fmt.Printf("  id=%d → not found (wrapped: %v)\n", id, err)
		case errors.Is(err, ErrPermission):
			fmt.Printf("  id=%d → access denied\n", id)
		default:
			fmt.Printf("  id=%d → unexpected error: %v\n", id, err)
		}
	}

	// Demonstrate that errors.Is walks the chain.
	wrapped := fmt.Errorf("service layer: %w", fmt.Errorf("repo layer: %w", ErrNotFound))
	fmt.Println("\n  chain:", wrapped)
	fmt.Println("  errors.Is(wrapped, ErrNotFound):", errors.Is(wrapped, ErrNotFound)) // true
}
