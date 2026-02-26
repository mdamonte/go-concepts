package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
)

// httptest.NewRecorder — unit-test a handler with zero network overhead.
// httptest.NewServer   — integration-test with a real TCP listener.
//
// When to use which:
//   NewRecorder  → fast, isolated handler tests; no OS resources needed
//   NewServer    → test the full HTTP client + server roundtrip (e.g., your http.Client code)

// userHandler is the handler under test.
func userHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		id := r.PathValue("id")
		if id == "" {
			http.Error(w, "missing id", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"id": id, "name": "Alice"})

	case http.MethodPost:
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(body)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func demoRecorder() {
	// ── httptest.NewRecorder ──────────────────────────────────────────────────
	// Simulates an http.ResponseWriter without any network.
	// Call handler directly — no server needed.
	fmt.Println("  httptest.NewRecorder — zero network, direct handler call:")

	// Register GET /users/{id} and POST /users as separate routes
	mux := http.NewServeMux()
	mux.HandleFunc("GET /users/{id}", userHandler)
	mux.HandleFunc("POST /users", userHandler)

	// Success: GET /users/42
	req := httptest.NewRequest(http.MethodGet, "/users/42", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	res := w.Result()
	defer res.Body.Close()
	fmt.Printf("  GET /users/42        → %d  Content-Type: %s\n",
		res.StatusCode, res.Header.Get("Content-Type"))
	fmt.Printf("  body: %s\n", strings.TrimSpace(w.Body.String()))

	// Error: POST /users with invalid JSON → 400
	req2 := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(`not-json`))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	mux.ServeHTTP(w2, req2)
	fmt.Printf("\n  POST /users (bad JSON) → %d\n", w2.Code)

	// Success: POST /users with valid JSON → 201
	req3 := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(`{"name":"Bob"}`))
	req3.Header.Set("Content-Type", "application/json")
	w3 := httptest.NewRecorder()
	mux.ServeHTTP(w3, req3)
	fmt.Printf("\n  POST /users (valid)    → %d %s\n", w3.Code, strings.TrimSpace(w3.Body.String()))

	// ── httptest.NewServer ────────────────────────────────────────────────────
	// Starts a real TCP server — use when testing an http.Client or middleware
	// that must make real network calls.
	fmt.Println("\n  httptest.NewServer — real TCP, for testing clients:")
	srv := httptest.NewServer(mux)
	defer srv.Close()
	fmt.Printf("  server URL: %s\n", srv.URL)

	// Use a real http.Client to call the test server
	client := &http.Client{}
	resp, _ := client.Get(srv.URL + "/users/99")
	defer resp.Body.Close()
	fmt.Printf("  GET /users/99 via http.Client  → %d\n", resp.StatusCode)

	// ── What a _test.go file looks like ──────────────────────────────────────
	// In a real codebase, these patterns live in *_test.go files:
	//
	//   func TestUserHandlerGet(t *testing.T) {
	//       req := httptest.NewRequest("GET", "/users/1", nil)
	//       w   := httptest.NewRecorder()
	//       userHandler(w, req)
	//
	//       if w.Code != http.StatusOK {
	//           t.Errorf("want 200, got %d", w.Code)
	//       }
	//       // assert body, headers, etc.
	//   }
	//
	//   // Table-driven:
	//   func TestUserHandler(t *testing.T) {
	//       tests := []struct {
	//           method string
	//           path   string
	//           body   string
	//           want   int
	//       }{
	//           {"GET",    "/users/1",  "",             200},
	//           {"GET",    "/users/",   "",             400},
	//           {"POST",   "/users/",   `{"name":"X"}`, 201},
	//           {"DELETE", "/users/1",  "",             405},
	//       }
	//       for _, tc := range tests {
	//           t.Run(fmt.Sprintf("%s %s", tc.method, tc.path), func(t *testing.T) {
	//               req := httptest.NewRequest(tc.method, tc.path,
	//                   strings.NewReader(tc.body))
	//               w := httptest.NewRecorder()
	//               mux.ServeHTTP(w, req)
	//               if w.Code != tc.want {
	//                   t.Errorf("want %d, got %d", tc.want, w.Code)
	//               }
	//           })
	//       }
	//   }
	fmt.Println("\n  (see comments in recorder.go for _test.go table-driven pattern)")
}
