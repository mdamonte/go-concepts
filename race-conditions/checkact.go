package main

import (
	"fmt"
	"sync"
)

// Check-then-act (TOCTOU: Time Of Check To Time Of Use) is a race where
// a goroutine reads a condition, decides to act, but the condition changes
// before the action is performed — invalidating the original decision.
//
// Classic example: two goroutines withdraw from the same account concurrently.
// Both check the balance (100), both see it's sufficient, both withdraw 100 —
// the account ends up at -100.

type account struct {
	balance int
}

// withdrawRacy reads the balance and deducts in two separate steps.
// Between the check and the deduct, another goroutine can run its own check
// and see the same (unmodified) balance — both withdrawals succeed.
func (a *account) withdrawRacy(amount int) bool {
	if a.balance >= amount { // CHECK — balance looks sufficient
		// ← another goroutine can run here and also pass the check
		a.balance -= amount // ACT — balance is now negative
		return true
	}
	return false
}

// demoCheckActRace launches 10 goroutines, each trying to withdraw 100
// from an account with a balance of 100. Without synchronization, multiple
// withdrawals succeed and the balance goes negative.
func demoCheckActRace() {
	a := &account{balance: 100}
	var wg sync.WaitGroup
	successes := 0
	var mu sync.Mutex // only to safely count successes for printing

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if a.withdrawRacy(100) { // DATA RACE on a.balance
				mu.Lock()
				successes++
				mu.Unlock()
			}
		}()
	}

	wg.Wait()
	fmt.Printf("  balance: %d  successful withdrawals: %d  (expected balance ≥ 0)\n",
		a.balance, successes)
}

// ── Fixed version ─────────────────────────────────────────────────────────────

type safeAccount struct {
	mu      sync.Mutex
	balance int
}

// withdraw holds the lock across the entire check-and-act sequence.
// No other goroutine can sneak in between the read and the write.
func (a *safeAccount) withdraw(amount int) bool {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.balance >= amount { // check
		a.balance -= amount  // act — atomic with respect to other goroutines
		return true
	}
	return false
}

func (a *safeAccount) deposit(amount int) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.balance += amount
}

// demoCheckActFixed shows that with the lock spanning the full check+act,
// at most one withdrawal can succeed on a balance of 100.
func demoCheckActFixed() {
	a := &safeAccount{balance: 100}
	var wg sync.WaitGroup
	var mu sync.Mutex
	successes := 0

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if a.withdraw(100) {
				mu.Lock()
				successes++
				mu.Unlock()
			}
		}()
	}

	wg.Wait()
	fmt.Printf("  balance: %d  successful withdrawals: %d  ✓\n",
		a.balance, successes) // balance: 0, withdrawals: 1
}
