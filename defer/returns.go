package main

import (
	"errors"
	"fmt"
)

// Named vs anonymous return values with defer — the most-asked senior question.
//
// Mental model:
//   return X   is really two steps:
//     1. assign X to the return slot (named → uses the named var; anonymous → copies X)
//     2. run deferred functions
//     3. return to caller
//
// A deferred function that modifies a NAMED return variable changes what the
// caller receives. A deferred function that modifies a LOCAL variable does not.

// ── Anonymous return — defer cannot change what the caller sees ───────────────
func anonymousReturn() int {
	x := 5
	defer func() {
		x *= 2 // modifies local x — the return slot already holds 5
	}()
	return x // step 1: copy x (=5) into return slot; step 2: defer runs; step 3: return 5
}

// ── Named return — defer CAN change what the caller sees ─────────────────────
func namedReturn() (result int) {
	defer func() {
		result *= 2 // modifies the actual return variable
	}()
	result = 5
	return // step 1: result is already 5; step 2: defer sets result=10; step 3: return 10
}

// ── Named return with explicit value — same trap ──────────────────────────────
func namedReturnExplicit() (result int) {
	defer func() {
		result *= 2
	}()
	return 5 // "return 5" assigns result=5, THEN defer runs, result becomes 10
}

// ── Practical use 1: annotate errors with context ────────────────────────────
// Instead of wrapping every early return manually, defer does it once.

var ErrNotFound = errors.New("not found")

func findRecord(id int) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("findRecord(%d): %w", id, err) // wrap with context
		}
	}()

	if id <= 0 {
		return fmt.Errorf("invalid id %d", id)
	}
	if id > 100 {
		return ErrNotFound // every early return gets wrapped automatically
	}
	return nil
}

// ── Practical use 2: transaction rollback / commit ────────────────────────────
// The standard Go pattern for database transactions.

type tx struct{ committed bool }

func (t *tx) Rollback() { fmt.Println("  tx: rollback") }
func (t *tx) Commit() error {
	t.committed = true
	fmt.Println("  tx: commit")
	return nil
}

func beginTx() (*tx, error) { return &tx{}, nil }

// withTx runs fn inside a transaction.
// If fn returns an error (or panics), the transaction is rolled back.
// Otherwise it is committed. The named return `err` lets defer inspect
// the final error value, including any panic-turned-error.
func withTx(fn func(*tx) error) (err error) {
	t, err := beginTx()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			t.Rollback()
		} else {
			err = t.Commit() // commit error is returned to caller via named `err`
		}
	}()
	return fn(t)
}

func demoReturns() {
	fmt.Println("  anonymousReturn() — defer modifies local, caller gets 5:")
	fmt.Println("  result =", anonymousReturn()) // 5

	fmt.Println("\n  namedReturn() — defer modifies named var, caller gets 10:")
	fmt.Println("  result =", namedReturn()) // 10

	fmt.Println("\n  namedReturnExplicit() — 'return 5' sets result=5, defer doubles it:")
	fmt.Println("  result =", namedReturnExplicit()) // 10

	fmt.Println("\n  findRecord — defer wraps every error with context:")
	for _, id := range []int{-1, 200, 42} {
		err := findRecord(id)
		if err != nil {
			fmt.Printf("  findRecord(%3d) → %v\n", id, err)
		} else {
			fmt.Printf("  findRecord(%3d) → ok\n", id)
		}
	}
	fmt.Println("  errors.Is still works through the wrap:")
	fmt.Println("  Is(ErrNotFound):", errors.Is(findRecord(200), ErrNotFound))

	fmt.Println("\n  withTx — commit on success, rollback on error:")
	_ = withTx(func(t *tx) error {
		fmt.Println("  fn: doing work — no error")
		return nil // → commit
	})
	_ = withTx(func(t *tx) error {
		fmt.Println("  fn: doing work — error!")
		return errors.New("constraint violation") // → rollback
	})
}
