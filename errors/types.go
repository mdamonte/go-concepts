package main

import (
	"errors"
	"fmt"
	"net/http"
)

// ValidationError is a custom error type that carries structured data.
// Implementing the error interface requires only the Error() string method.
//
// Use a custom type (instead of a sentinel) when the error needs to carry
// fields that callers can inspect programmatically.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error on %q: %s", e.Field, e.Message)
}

// HTTPError wraps an HTTP status code with a message.
type HTTPError struct {
	Code    int
	Message string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("HTTP %d: %s", e.Code, e.Message)
}

// parseAge simulates input validation that returns a *ValidationError.
func parseAge(s string) (int, error) {
	if s == "" {
		return 0, &ValidationError{Field: "age", Message: "must not be empty"}
	}
	if s == "abc" {
		return 0, &ValidationError{Field: "age", Message: "must be a number"}
	}
	return 30, nil
}

// fetchResource simulates an HTTP call that may fail with an *HTTPError.
func fetchResource(path string) error {
	if path == "/secret" {
		return fmt.Errorf("fetch %q: %w", path, &HTTPError{Code: http.StatusForbidden, Message: "forbidden"})
	}
	if path == "/missing" {
		return fmt.Errorf("fetch %q: %w", path, &HTTPError{Code: http.StatusNotFound, Message: "not found"})
	}
	return nil
}

// demoCustomType shows how to define custom error types and use errors.As
// to extract the concrete type from the error chain — even when wrapped.
//
// errors.As walks the Unwrap chain looking for a value assignable to the
// target pointer. It is the typed equivalent of errors.Is.
func demoCustomType() {
	// ── ValidationError ──────────────────────────────────────────────────────
	inputs := []string{"", "abc", "25"}
	for _, input := range inputs {
		_, err := parseAge(input)
		if err == nil {
			fmt.Printf("  parseAge(%q) → ok\n", input)
			continue
		}

		var valErr *ValidationError
		if errors.As(err, &valErr) {
			fmt.Printf("  parseAge(%q) → field=%q msg=%q\n", input, valErr.Field, valErr.Message)
		}
	}

	// ── HTTPError (wrapped) ───────────────────────────────────────────────────
	fmt.Println()
	paths := []string{"/public", "/secret", "/missing"}
	for _, path := range paths {
		err := fetchResource(path)
		if err == nil {
			fmt.Printf("  fetch(%q) → ok\n", path)
			continue
		}

		var httpErr *HTTPError
		if errors.As(err, &httpErr) {
			// errors.As unwrapped the chain to find *HTTPError.
			fmt.Printf("  fetch(%q) → code=%d msg=%q\n", path, httpErr.Code, httpErr.Message)
		}
	}
}
