package main

import (
	"context"
	"fmt"
	"time"
)

// demoDeadline shows WithDeadline: the context cancels at an absolute time.Time.
//
// Use WithDeadline when you have a wall-clock SLA (e.g. "respond by 14:00:05").
// Use WithTimeout when you have a relative budget (e.g. "spend at most 500 ms").
// Under the hood, WithTimeout(ctx, d) is sugar for WithDeadline(ctx, time.Now().Add(d)).
func demoDeadline() {
	abs := time.Now().Add(150 * time.Millisecond)
	ctx, cancel := context.WithDeadline(context.Background(), abs)
	defer cancel()

	fmt.Printf("deadline set to: %s\n", abs.Format("15:04:05.000"))

	// Simulate work that exceeds the deadline.
	select {
	case <-time.After(500 * time.Millisecond):
		fmt.Println("work done (unreachable)")
	case <-ctx.Done():
		fmt.Printf("deadline fired at: %s → %v\n",
			time.Now().Format("15:04:05.000"), ctx.Err())
	}

	// Calling cancel() after the deadline has already fired is a no-op,
	// but always deferring it ensures we don't leak the timer goroutine
	// in the case where we return before the deadline.
}
