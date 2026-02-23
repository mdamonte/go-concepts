package main

import (
	"context"
	"fmt"
)

// Use unexported, package-local types for context keys.
// This prevents key collisions between packages that happen to use the same string.
// Never use plain string / int literals as keys directly.
type ctxKey string

const (
	keyRequestID ctxKey = "requestID"
	keyUserID    ctxKey = "userID"
)

// demoValue shows how to thread request-scoped data down a call chain.
// Good candidates: request IDs, auth tokens, trace IDs, logger instances.
// Bad candidates: optional function parameters, large objects, mutable state.
func demoValue() {
	// Values are immutable: each WithValue wraps the parent and adds one pair.
	ctx := context.Background()
	ctx = context.WithValue(ctx, keyRequestID, "req-abc-123")
	ctx = context.WithValue(ctx, keyUserID, 42)

	handleRequest(ctx)
}

func handleRequest(ctx context.Context) {
	// Type-assert the value; always check the ok flag to avoid panics.
	reqID, ok := ctx.Value(keyRequestID).(string)
	if !ok {
		reqID = "unknown"
	}
	userID, _ := ctx.Value(keyUserID).(int)

	fmt.Printf("handleRequest  → reqID=%s  userID=%d\n", reqID, userID)

	// Pass the same ctx down; no need to re-attach the values.
	processRequest(ctx)
}

func processRequest(ctx context.Context) {
	reqID := ctx.Value(keyRequestID).(string)
	fmt.Printf("processRequest → reqID=%s (value flows transparently)\n", reqID)
}
