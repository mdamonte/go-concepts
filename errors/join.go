package main

import (
	"errors"
	"fmt"
)

// demoJoin shows errors.Join (Go 1.20+): combine multiple errors into one.
//
// errors.Join returns nil if all inputs are nil.
// The resulting error's Error() joins non-nil messages with newlines.
// errors.Is and errors.As work against every error in the joined set.
func demoJoin() {
	// ── Basic join ────────────────────────────────────────────────────────────
	err1 := errors.New("database timeout")
	err2 := errors.New("cache miss")
	err3 := errors.New("rate limit exceeded")

	joined := errors.Join(err1, err2, err3)
	fmt.Println("  joined error:")
	fmt.Println(" ", joined)

	// errors.Is checks all branches of the joined error tree.
	fmt.Println("  Is(err2):", errors.Is(joined, err2)) // true

	// nil inputs are ignored; all-nil returns nil.
	noErr := errors.Join(nil, nil)
	fmt.Println("  Join(nil, nil):", noErr) // <nil>

	// ── Collecting errors from concurrent work ────────────────────────────────
	// Simulate validating multiple fields and collecting all failures at once.
	type field struct {
		name  string
		value string
	}
	fields := []field{
		{"username", ""},
		{"email", "not-an-email"},
		{"age", "25"},
	}

	var errs []error
	for _, f := range fields {
		if f.value == "" {
			errs = append(errs, &ValidationError{Field: f.name, Message: "must not be empty"})
		} else if f.name == "email" {
			errs = append(errs, &ValidationError{Field: f.name, Message: "invalid format"})
		}
	}

	if combined := errors.Join(errs...); combined != nil {
		fmt.Printf("\n  validation collected %d error(s):\n", len(errs))
		fmt.Println(" ", combined)

		// errors.As still finds the first matching type in the tree.
		var ve *ValidationError
		if errors.As(combined, &ve) {
			fmt.Printf("  first ValidationError: field=%q\n", ve.Field)
		}
	}
}
