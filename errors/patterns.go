package main

import (
	"errors"
	"fmt"
)

// ── Patrón: error de operación con contexto ───────────────────────────────────

// OpError es el patrón de error de la stdlib (net.OpError, os.PathError):
// captura la operación, el recurso y la causa para mensajes ricos sin perder
// la capacidad de inspeccionar la causa con errors.Is/As.
type OpError struct {
	Op   string // "read", "write", "connect", …
	Path string // resource identifier
	Err  error  // underlying cause
}

func (e *OpError) Error() string {
	return fmt.Sprintf("%s %s: %v", e.Op, e.Path, e.Err)
}

// Unwrap exposes the underlying error to errors.Is and errors.As.
func (e *OpError) Unwrap() error { return e.Err }

var ErrConnectionRefused = errors.New("connection refused")

func openDB(dsn string) error {
	// Simulate a failed connection attempt.
	return &OpError{Op: "connect", Path: dsn, Err: ErrConnectionRefused}
}

func demoOpError() {
	err := openDB("postgres://localhost:5432/mydb")
	fmt.Println("  error:", err)

	// errors.Is traverses via Unwrap().
	fmt.Println("  Is(ErrConnectionRefused):", errors.Is(err, ErrConnectionRefused))

	// errors.As extracts the *OpError.
	var opErr *OpError
	if errors.As(err, &opErr) {
		fmt.Printf("  op=%q path=%q\n", opErr.Op, opErr.Path)
	}
}

// ── Patrón: errores opacos vs exportados ────────────────────────────────────

// Exported sentinel — callers CAN check for it.
var ErrInvalidInput = errors.New("invalid input")

// opaqueErr is unexported — callers CANNOT check for it specifically.
// Use this when the error is an internal implementation detail.
var opaqueErr = errors.New("internal state corrupted")

func process(input string) error {
	if input == "" {
		// Exported: callers are expected to handle this case.
		return fmt.Errorf("process: %w", ErrInvalidInput)
	}
	if input == "corrupt" {
		// Opaque: wrap with %v so the chain is broken intentionally.
		// The caller gets a message but cannot match the internal error.
		return fmt.Errorf("process: %v", opaqueErr)
	}
	return nil
}

func demoOpaque() {
	cases := []string{"", "corrupt", "ok"}
	for _, input := range cases {
		err := process(input)
		if err == nil {
			fmt.Printf("  process(%q) → ok\n", input)
			continue
		}
		fmt.Printf("  process(%q) → %v\n", input, err)
		fmt.Printf("    Is(ErrInvalidInput): %v\n", errors.Is(err, ErrInvalidInput))
	}
	// Output shows that only the exported sentinel is detectable by callers.
}

// ── Patrón: panic vs error ───────────────────────────────────────────────────

// demoP anicVsError illustrates the Go convention:
//
//   - Return errors for expected, recoverable conditions (missing file,
//     bad input, network failure, …).
//   - Panic for programming mistakes that should never happen in correct code
//     (nil pointer on a required dep, index out of bounds, invariant violation).
//
// A panic unwinds the stack and, if not recovered, crashes the program.
// It should NEVER cross a public API boundary — convert it to an error there.
func demoPanicVsError() {
	// ── Recover at API boundary ───────────────────────────────────────────────
	safeDiv := func(a, b int) (result int, err error) {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("safeDiv: panic: %v", r)
			}
		}()
		result = a / b // panics if b == 0
		return
	}

	if v, err := safeDiv(10, 2); err == nil {
		fmt.Printf("  10 / 2 = %d\n", v)
	}
	if _, err := safeDiv(10, 0); err != nil {
		fmt.Printf("  10 / 0 → recovered: %v\n", err)
	}

	// ── When to panic ─────────────────────────────────────────────────────────
	// Panic is appropriate for programmer errors caught at startup.
	mustParseConfig := func(path string) string {
		if path == "" {
			// This is a programming mistake — fail fast, not silently.
			panic("mustParseConfig: path must not be empty")
		}
		return path
	}

	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("  mustParseConfig caught panic: %v\n", r)
		}
	}()
	mustParseConfig("") // intentional panic to demonstrate recovery
}
