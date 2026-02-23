package main

import (
	"context"
	"fmt"
	"time"
)

// demoTimeout shows WithTimeout: the context cancels automatically after a duration.
// ctx.Err() returns context.DeadlineExceeded when the timeout fires.
func demoTimeout() {
	// Case 1: work finishes before the timeout → no error.
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel() // still required: frees resources if we return before the timeout

	err := fakeHTTPCall(ctx, 50*time.Millisecond)
	fmt.Println("fast call (50ms, timeout 300ms):", orOK(err))

	// Case 2: work takes longer than the timeout → DeadlineExceeded.
	ctx2, cancel2 := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel2()

	err = fakeHTTPCall(ctx2, 300*time.Millisecond)
	fmt.Println("slow call (300ms, timeout 50ms): ", orOK(err))

	// Case 3: check remaining time before starting work.
	ctx3, cancel3 := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel3()

	deadline, ok := ctx3.Deadline()
	if ok {
		fmt.Printf("time until deadline: %v\n", time.Until(deadline).Round(time.Millisecond))
	}
}

// fakeHTTPCall simulates an outbound call that respects context cancellation.
func fakeHTTPCall(ctx context.Context, latency time.Duration) error {
	select {
	case <-time.After(latency):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func orOK(err error) string {
	if err != nil {
		return err.Error()
	}
	return "ok"
}
