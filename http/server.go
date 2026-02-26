package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
)

// ── http.Handler — the core interface ────────────────────────────────────────
//
//	type Handler interface {
//	    ServeHTTP(ResponseWriter, *Request)
//	}
//
// Any type that implements ServeHTTP is a Handler.
// http.HandlerFunc is an adapter that turns a plain function into a Handler.

// greetHandler is a struct-based handler — useful when the handler needs state.
type greetHandler struct{ greeting string }

func (h greetHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	if name == "" {
		name = "world"
	}
	fmt.Fprintf(w, "%s, %s!\n", h.greeting, name)
}

// ── http.ServeMux — request router ───────────────────────────────────────────
// Go 1.22 enhanced ServeMux with method prefixes and {wildcard} patterns.
// Before 1.22: only path matching, no method routing in stdlib.
//
// Pattern precedence (most specific wins):
//   "GET /users/me"    beats   "GET /users/{id}"
//   "/users/"          beats   "/"
//
// Avoid http.DefaultServeMux in production — third-party packages can
// accidentally register routes on it via init().

func newRouter() *http.ServeMux {
	mux := http.NewServeMux()

	// Struct-based handler
	mux.Handle("GET /greet", greetHandler{greeting: "Hello"})

	// Function handler — Go 1.22 method prefix
	mux.HandleFunc("GET /users/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id") // Go 1.22: extract {id} from the path
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"id": id, "name": "Alice"})
	})

	// POST — read JSON body
	mux.HandleFunc("POST /users", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(body)
	})

	// Raw body read + echo
	mux.HandleFunc("POST /echo", func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "read error", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()
		w.Write(body)
	})

	// 204 No Content — DELETE pattern
	mux.HandleFunc("DELETE /users/{id}", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent) // must call before writing body
	})

	// Catch-all wildcard {path...}
	mux.HandleFunc("/files/{path...}", func(w http.ResponseWriter, r *http.Request) {
		path := r.PathValue("path")
		fmt.Fprintf(w, "serving file: %s\n", path)
	})

	return mux
}

func demoServer() {
	srv := httptest.NewServer(newRouter())
	defer srv.Close()
	fmt.Printf("  test server at %s\n\n", srv.URL)

	get := func(path string) {
		resp, err := http.Get(srv.URL + path)
		if err != nil {
			fmt.Printf("  GET %-30s → error: %v\n", path, err)
			return
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		fmt.Printf("  GET %-30s → %d %s\n", path, resp.StatusCode, strings.TrimSpace(string(body)))
	}

	get("/greet?name=Gopher")
	get("/users/42")
	get("/files/assets/logo.png")

	// POST with JSON body
	resp, _ := http.Post(srv.URL+"/users", "application/json",
		strings.NewReader(`{"name":"Bob","age":30}`))
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	fmt.Printf("  POST /users                    → %d %s\n", resp.StatusCode, strings.TrimSpace(string(body)))

	// DELETE — 204 No Content
	req, _ := http.NewRequest(http.MethodDelete, srv.URL+"/users/42", nil)
	resp, _ = http.DefaultClient.Do(req)
	resp.Body.Close()
	fmt.Printf("  DELETE /users/42               → %d (No Content)\n", resp.StatusCode)
}
