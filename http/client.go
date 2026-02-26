package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"
)

// ── http.Client — always use a custom client ──────────────────────────────────
// http.DefaultClient has no timeout. A server that never responds will block
// the goroutine forever, leaking resources.
//
//   http.DefaultClient          ← no timeout — never use in production
//   &http.Client{Timeout: 5s}   ← always set a Timeout

// ── Always check status code explicitly ──────────────────────────────────────
// http.Client does NOT return an error for non-2xx responses.
// A 404 or 500 is a successful HTTP exchange from the transport's perspective.
// The error return only signals transport-level failures (DNS, TLS, timeout).

// ── Always read and close the response body ──────────────────────────────────
// Even when you don't care about the body, you must drain and close it.
// Otherwise the underlying TCP connection cannot be reused by the transport.
//
//   defer resp.Body.Close()          // always close
//   io.Copy(io.Discard, resp.Body)   // drain if you don't read the full body

func demoClient() {
	// Test server with fast, slow, and error endpoints
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/fast":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		case "/slow":
			time.Sleep(300 * time.Millisecond)
			fmt.Fprintln(w, "finally done")
		case "/error":
			http.Error(w, "something went wrong", http.StatusInternalServerError)
		case "/echo":
			io.Copy(w, r.Body)
		}
	}))
	defer srv.Close()

	client := &http.Client{Timeout: 5 * time.Second}

	// ── Successful JSON response ──────────────────────────────────────────────
	fmt.Println("  Custom client (Timeout: 5s):")
	resp, err := client.Get(srv.URL + "/fast")
	if err != nil {
		fmt.Println("  transport error:", err)
		return
	}
	defer resp.Body.Close()
	var result map[string]string
	json.NewDecoder(resp.Body).Decode(&result)
	fmt.Printf("  GET /fast → %d %v\n", resp.StatusCode, result)

	// ── Non-2xx status — no error from client.Do ──────────────────────────────
	fmt.Println("\n  Non-2xx does NOT return an error — check status explicitly:")
	resp2, err := client.Get(srv.URL + "/error")
	if err != nil {
		fmt.Println("  transport error:", err)
		return
	}
	defer resp2.Body.Close()
	if resp2.StatusCode >= 400 {
		body, _ := io.ReadAll(resp2.Body)
		fmt.Printf("  GET /error → %d %q\n", resp2.StatusCode, strings.TrimSpace(string(body)))
	}

	// ── Context cancellation — abort in-flight request ────────────────────────
	// http.NewRequestWithContext attaches a context at construction time.
	// Prefer it over req.WithContext(ctx) — the latter returns a shallow copy
	// and is easy to misuse.
	fmt.Println("\n  Context cancellation — 50ms timeout on a 300ms endpoint:")
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+"/slow", nil)
	_, err = client.Do(req)
	if err != nil {
		fmt.Printf("  GET /slow cancelled: %v\n", err)
	}

	// ── POST with JSON body ───────────────────────────────────────────────────
	fmt.Println("\n  POST with JSON body:")
	ctx2, cancel2 := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel2()

	payload := strings.NewReader(`{"name":"Gopher","version":2}`)
	req2, _ := http.NewRequestWithContext(ctx2, http.MethodPost, srv.URL+"/echo", payload)
	req2.Header.Set("Content-Type", "application/json")

	resp3, err := client.Do(req2)
	if err != nil {
		fmt.Println("  error:", err)
		return
	}
	defer resp3.Body.Close()
	body3, _ := io.ReadAll(resp3.Body)
	fmt.Printf("  POST /echo → %d %s\n", resp3.StatusCode, strings.TrimSpace(string(body3)))

	// ── Drain body before closing (connection reuse) ──────────────────────────
	fmt.Println("\n  Drain body to allow connection reuse:")
	resp4, _ := client.Get(srv.URL + "/fast")
	io.Copy(io.Discard, resp4.Body) // drain before Close
	resp4.Body.Close()
	fmt.Println("  connection returned to pool")
}
