package main

import (
	"errors"
	"fmt"
)

// ── Custom Is() ──────────────────────────────────────────────────────────────

// StatusError represents an HTTP-like status error. Two StatusErrors are
// considered equal if their codes match, regardless of message.
type StatusError struct {
	Code    int
	Message string
}

func (e *StatusError) Error() string {
	return fmt.Sprintf("status %d: %s", e.Code, e.Message)
}

// Is makes errors.Is(target, StatusError{Code:404}) match any StatusError
// with Code 404, ignoring the message.
//
// Without this method, errors.Is uses == which requires the exact same pointer.
func (e *StatusError) Is(target error) bool {
	var t *StatusError
	if !errors.As(target, &t) {
		return false
	}
	return e.Code == t.Code
}

// demoCustomIs shows that implementing Is() lets errors.Is match on
// semantic equality rather than pointer identity.
func demoCustomIs() {
	sentinel := &StatusError{Code: 404, Message: ""}

	err1 := &StatusError{Code: 404, Message: "user not found"}
	err2 := &StatusError{Code: 500, Message: "internal error"}
	wrapped := fmt.Errorf("handler: %w", &StatusError{Code: 404, Message: "post not found"})

	fmt.Println("  err1 Is 404 sentinel:", errors.Is(err1, sentinel))    // true  — same code
	fmt.Println("  err2 Is 404 sentinel:", errors.Is(err2, sentinel))    // false — different code
	fmt.Println("  wrapped Is 404 sentinel:", errors.Is(wrapped, sentinel)) // true  — unwrapped
}

// ── Custom As() ──────────────────────────────────────────────────────────────

// MultiError holds several errors. Its As() method lets callers extract any
// single error from the collection by type.
type MultiError struct {
	Errors []error
}

func (m *MultiError) Error() string {
	return fmt.Sprintf("%d errors occurred", len(m.Errors))
}

// As searches each contained error for one assignable to target.
// This lets errors.As dig into the collection.
func (m *MultiError) As(target any) bool {
	for _, err := range m.Errors {
		if errors.As(err, target) {
			return true
		}
	}
	return false
}

// demoCustomAs shows that implementing As() lets errors.As search inside
// aggregate or container error types.
func demoCustomAs() {
	multi := &MultiError{
		Errors: []error{
			fmt.Errorf("first: %w", &ValidationError{Field: "name", Message: "too short"}),
			&StatusError{Code: 422, Message: "unprocessable"},
		},
	}

	var valErr *ValidationError
	if errors.As(multi, &valErr) {
		fmt.Printf("  found ValidationError: field=%q msg=%q\n", valErr.Field, valErr.Message)
	}

	var statusErr *StatusError
	if errors.As(multi, &statusErr) {
		fmt.Printf("  found StatusError: code=%d msg=%q\n", statusErr.Code, statusErr.Message)
	}
}
