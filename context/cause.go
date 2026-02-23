package main

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// Sentinel errors used as cancellation causes.
var (
	errRateLimit    = errors.New("rate limit exceeded")
	errServiceDown  = errors.New("downstream service unavailable")
	errQuotaReached = errors.New("monthly quota reached")
)

// demoCause shows the cause-aware constructors added in Go 1.20 / 1.21.
//
// Problem with plain WithCancel/WithTimeout: ctx.Err() can only return
// context.Canceled or context.DeadlineExceeded — you can't tell *why*
// the operation was cancelled.
//
// Solution: WithCancelCause / WithTimeoutCause / WithDeadlineCause let you
// attach a specific error. Retrieve it with context.Cause(ctx).
func demoCause() {
	demoCancelCause()
	demoTimeoutCause()
	demoDeadlineCause()
}

// WithCancelCause (Go 1.20): manual cancel with a reason.
func demoCancelCause() {
	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(nil) // nil = no extra cause; ctx.Err() will be Canceled

	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel(errRateLimit) // pass the specific reason
	}()

	<-ctx.Done()
	fmt.Printf("CancelCause  → ctx.Err(): %-26v  cause: %v\n",
		ctx.Err(), context.Cause(ctx))
	// ctx.Err()        → context.Canceled   (always)
	// context.Cause()  → errRateLimit        (the why)
}

// WithTimeoutCause (Go 1.21): automatic timeout with a cause on expiry.
func demoTimeoutCause() {
	ctx, cancel := context.WithTimeoutCause(
		context.Background(),
		80*time.Millisecond,
		errServiceDown, // attached only if the timeout fires
	)
	defer cancel()

	<-ctx.Done()
	fmt.Printf("TimeoutCause → ctx.Err(): %-26v  cause: %v\n",
		ctx.Err(), context.Cause(ctx))
	// ctx.Err()        → context.DeadlineExceeded
	// context.Cause()  → errServiceDown
}

// WithDeadlineCause (Go 1.21): absolute deadline with a cause on expiry.
func demoDeadlineCause() {
	abs := time.Now().Add(80 * time.Millisecond)
	ctx, cancel := context.WithDeadlineCause(
		context.Background(),
		abs,
		errQuotaReached,
	)
	defer cancel()

	<-ctx.Done()
	fmt.Printf("DeadlineCause→ ctx.Err(): %-26v  cause: %v\n",
		ctx.Err(), context.Cause(ctx))
}
