package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"
)

// Graceful shutdown — let in-flight requests finish before stopping.
//
// Pattern:
//
//  1. Start server in a goroutine; ListenAndServe blocks until Shutdown is called.
//  2. Wait for OS signal (in production: signal.NotifyContext):
//     ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
//     defer stop()
//     <-ctx.Done()
//  3. Call srv.Shutdown(shutdownCtx) — stops accepting new connections and
//     waits for active handlers to finish (up to shutdownCtx deadline).
//  4. srv.Serve / ListenAndServe returns http.ErrServerClosed — this is
//     expected and must NOT be treated as an error.

func demoShutdown() {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/slow" {
			time.Sleep(80 * time.Millisecond) // simulate in-flight work
		}
		fmt.Fprintln(w, "done")
	})

	// Use a random port to avoid conflicts with other demos
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		fmt.Println("  listen error:", err)
		return
	}
	addr := "http://" + ln.Addr().String()

	srv := &http.Server{Handler: handler}

	// Channel to collect the error from Serve
	serveErr := make(chan error, 1)
	go func() {
		fmt.Printf("  server listening at %s\n", addr)
		serveErr <- srv.Serve(ln)
	}()

	// Fire a slow request BEFORE shutdown — it must complete cleanly
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		time.Sleep(10 * time.Millisecond) // wait for server to be ready

		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Get(addr + "/slow")
		if err != nil {
			fmt.Println("  slow request error:", err)
			return
		}
		resp.Body.Close()
		fmt.Printf("  in-flight slow request completed: %d\n", resp.StatusCode)
	}()

	// Simulate receiving SIGINT after 30ms
	time.Sleep(30 * time.Millisecond)
	fmt.Println("  shutdown signal — draining in-flight requests...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		fmt.Println("  shutdown error:", err)
	} else {
		fmt.Println("  server shut down cleanly")
	}

	// srv.Serve returns http.ErrServerClosed — treat as success
	if err := <-serveErr; err != nil && !errors.Is(err, http.ErrServerClosed) {
		fmt.Println("  unexpected server error:", err)
	}

	wg.Wait() // wait for the slow request goroutine to finish printing
}
