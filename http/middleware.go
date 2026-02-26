package main

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"
)

// ── Middleware signature ──────────────────────────────────────────────────────
// A middleware wraps a Handler to add cross-cutting behaviour.
//
//	type Middleware func(http.Handler) http.Handler
//
// Execution order with Chain(h, mw1, mw2, mw3):
//
//	request  → mw1 → mw2 → mw3 → handler
//	response → mw3 → mw2 → mw1

// responseRecorder captures the status code written by a downstream handler.
// Embedding http.ResponseWriter promotes all its methods automatically.
type responseRecorder struct {
	http.ResponseWriter
	status int
}

func (r *responseRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

// Logger logs method, path, duration, and status code.
func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &responseRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		fmt.Printf("  [logger] %s %s → %d (%s)\n",
			r.Method, r.URL.Path, rec.status, time.Since(start).Round(time.Millisecond))
	})
}

// Auth requires a valid Bearer token in the Authorization header.
// Configured via closure — the token is captured at construction time.
func Auth(validToken string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			got := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
			if got != validToken {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return // do NOT call next
			}
			next.ServeHTTP(w, r)
		})
	}
}

// Recovery catches panics in downstream handlers and returns 500.
// Without this, a panicking handler crashes the goroutine serving the request.
func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				fmt.Printf("  [recovery] caught panic: %v\n", err)
				http.Error(w, "internal server error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// Chain applies middlewares right-to-left so the first listed runs outermost.
//
//	Chain(h, mw1, mw2, mw3) ≡ mw1(mw2(mw3(h)))
func Chain(h http.Handler, mws ...func(http.Handler) http.Handler) http.Handler {
	for i := len(mws) - 1; i >= 0; i-- {
		h = mws[i](h)
	}
	return h
}

func demoMiddleware() {
	const secret = "super-secret"

	mux := http.NewServeMux()

	// /protected requires a valid token — Logger wraps Auth wraps Recovery wraps handler
	mux.Handle("GET /protected",
		Chain(
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprintln(w, "secret data")
			}),
			Logger, Auth(secret), Recovery,
		),
	)

	// /panic — Recovery catches the panic and returns 500
	mux.Handle("GET /panic",
		Chain(
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				panic("something went badly wrong")
			}),
			Logger, Recovery,
		),
	)

	// /public — only Logger
	mux.Handle("GET /public",
		Chain(
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprintln(w, "public data")
			}),
			Logger,
		),
	)

	srv := httptest.NewServer(mux)
	defer srv.Close()
	fmt.Printf("  test server at %s\n\n", srv.URL)

	// No token → 401
	resp, _ := http.Get(srv.URL + "/protected")
	resp.Body.Close()
	fmt.Printf("  GET /protected (no token)      → %d\n\n", resp.StatusCode)

	// Valid token → 200
	req, _ := http.NewRequest("GET", srv.URL+"/protected", nil)
	req.Header.Set("Authorization", "Bearer "+secret)
	resp, _ = http.DefaultClient.Do(req)
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	fmt.Printf("  GET /protected (valid token)   → %d %q\n\n", resp.StatusCode, strings.TrimSpace(string(body)))

	// Panic → 500 recovered
	resp, _ = http.Get(srv.URL + "/panic")
	resp.Body.Close()
	fmt.Printf("  GET /panic (recovered)         → %d\n\n", resp.StatusCode)

	// Public → 200
	resp, _ = http.Get(srv.URL + "/public")
	resp.Body.Close()
	fmt.Printf("  GET /public                    → %d\n", resp.StatusCode)
}
