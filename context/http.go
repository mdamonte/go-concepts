package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"time"
)

func demoHTTP() {
	demoClientTimeout()
	demoClientCancel()
	demoServerDisconnect()
	demoServerPropagation()
}

// ── Client side ───────────────────────────────────────────────────────────────

// demoClientTimeout shows how to set a deadline on an outbound HTTP request.
//
// http.NewRequestWithContext is the correct API (never http.NewRequest for
// production code — it has no way to be cancelled).
func demoClientTimeout() {
	fmt.Println("── client timeout ──")

	// Server that always takes 300 ms to respond.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(300 * time.Millisecond)
		fmt.Fprintln(w, "pong")
	}))
	defer srv.Close()

	// Case 1: timeout longer than server latency → success.
	{
		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()

		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL, nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			fmt.Println("  long timeout: error:", err)
		} else {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			fmt.Printf("  long timeout: %d %q\n", resp.StatusCode, string(body))
		}
	}

	// Case 2: timeout shorter than server latency → DeadlineExceeded.
	{
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL, nil)
		_, err := http.DefaultClient.Do(req)
		fmt.Println("  short timeout:", err)
	}
}

// demoClientCancel shows how to abort an in-flight request from another goroutine.
// Useful for "first response wins" fan-out patterns, user-initiated cancellations, etc.
func demoClientCancel() {
	fmt.Println("── client cancel ──")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-time.After(500 * time.Millisecond):
			fmt.Fprintln(w, "pong")
		case <-r.Context().Done():
			// Client disconnected; stop processing to avoid wasted work.
			return
		}
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL, nil)

	// Cancel the request after 80 ms from a separate goroutine.
	go func() {
		time.Sleep(80 * time.Millisecond)
		fmt.Println("  cancelling request...")
		cancel()
	}()

	_, err := http.DefaultClient.Do(req)
	fmt.Println("  after cancel:", err)
}

// ── Server side ───────────────────────────────────────────────────────────────

// demoServerDisconnect shows how a handler can detect that the client has
// gone away (e.g. browser tab closed, upstream proxy timeout) and abort
// expensive work early instead of computing a response nobody will read.
func demoServerDisconnect() {
	fmt.Println("── server detects client disconnect ──")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context() // request context is cancelled when client disconnects

		select {
		case <-time.After(500 * time.Millisecond):
			// Expensive work finished.
			fmt.Fprintln(w, "done")
		case <-ctx.Done():
			// Client is gone; no point writing the response.
			fmt.Println("  [server] client disconnected:", ctx.Err())
			// http.StatusServiceUnavailable or just return — the connection is dead.
			return
		}
	}))
	defer srv.Close()

	// Client that cancels after 80 ms (simulating a disconnect).
	ctx, cancel := context.WithTimeout(context.Background(), 80*time.Millisecond)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL, nil)
	_, err := http.DefaultClient.Do(req)
	fmt.Println("  [client] request ended:", err)

	time.Sleep(50 * time.Millisecond) // let the server log its cancellation
}

// demoServerPropagation shows the most important server-side pattern:
// pass r.Context() to every downstream call (DB, gRPC, another HTTP service)
// so the whole call chain is cancelled if the client disconnects or times out.
func demoServerPropagation() {
	fmt.Println("── server propagates context to downstream ──")

	// Downstream "database" service.
	db := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-time.After(200 * time.Millisecond):
			fmt.Fprintln(w, `{"user":"alice"}`)
		case <-r.Context().Done():
			http.Error(w, "cancelled", http.StatusServiceUnavailable)
		}
	}))
	defer db.Close()

	// Frontend handler that queries the downstream service using r.Context().
	frontend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// ✅ Correct: propagate the request context to the downstream call.
		// If the original client cancels, this sub-request is also cancelled.
		downstream, _ := http.NewRequestWithContext(r.Context(), http.MethodGet, db.URL, nil)
		resp, err := http.DefaultClient.Do(downstream)
		if err != nil {
			http.Error(w, "downstream failed: "+err.Error(), http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		fmt.Fprintf(w, "frontend got: %s", body)
	}))
	defer frontend.Close()

	// Happy path: client waits long enough.
	{
		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()

		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, frontend.URL, nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			fmt.Println("  propagation:", err)
		} else {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			fmt.Printf("  propagation: %d %s", resp.StatusCode, body)
		}
	}
}
