package main

import (
	"errors"
	"fmt"
)

// demoWrapping shows how fmt.Errorf with %w builds a chain of wrapped errors,
// and how errors.Unwrap, errors.Is and errors.As traverse that chain.
//
// Rules for wrapping:
//   - Use %w to wrap when the caller may need to inspect the cause.
//   - Use %v (or a plain string) when the cause is an implementation detail
//     that callers should NOT depend on.
//   - Wrap at each layer to add context; don't rewrap the same error.
//
// Message convention: "operation: cause" — no capital letter, no trailing dot.
func demoWrapping() {
	// Build a three-level chain: db → repo → service.
	dbErr := errors.New("connection refused")
	repoErr := fmt.Errorf("repo.FindUser: %w", dbErr)
	svcErr := fmt.Errorf("service.GetUser id=42: %w", repoErr)

	fmt.Println("  full error:", svcErr)
	// service.GetUser id=42: repo.FindUser: connection refused

	// errors.Unwrap peels exactly one layer.
	fmt.Println("  Unwrap once:", errors.Unwrap(svcErr))   // repo.FindUser: …
	fmt.Println("  Unwrap twice:", errors.Unwrap(errors.Unwrap(svcErr))) // connection refused

	// errors.Is walks the whole chain.
	fmt.Println("  Is(dbErr):", errors.Is(svcErr, dbErr)) // true

	// errors.As also walks the chain.
	var valErr *ValidationError
	wrapped := fmt.Errorf("outer: %w", &ValidationError{Field: "email", Message: "invalid format"})
	if errors.As(wrapped, &valErr) {
		fmt.Printf("  As(*ValidationError): field=%q msg=%q\n", valErr.Field, valErr.Message)
	}

	// ── %v vs %w ─────────────────────────────────────────────────────────────
	// Using %v hides the cause: errors.Is cannot find it.
	opaque := fmt.Errorf("something went wrong: %v", dbErr) // %v, not %w
	fmt.Println("\n  opaque error:", opaque)
	fmt.Println("  Is(dbErr) through %%v:", errors.Is(opaque, dbErr)) // false — chain is broken
}
