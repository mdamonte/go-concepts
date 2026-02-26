package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	_ "net/http/pprof" // side-effect: registers /debug/pprof/* on http.DefaultServeMux
	"time"
)

// demoHTTPPprof shows the always-on HTTP profiling endpoints.
//
// A single blank import registers all pprof routes on http.DefaultServeMux:
//
//	import _ "net/http/pprof"
//
// WARNING: never expose these endpoints on a public port.
// They leak internal details and allow triggering CPU/memory load.
// Bind to localhost or put behind authentication middleware.
//
// Mount on your own mux (instead of DefaultServeMux):
//
//	mux.HandleFunc("/debug/pprof/", pprof.Index)
//	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
//	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
//	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
//	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

func demoHTTPPprof() {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		fmt.Println("  listen error:", err)
		return
	}
	addr := ln.Addr().String()

	srv := &http.Server{} // uses http.DefaultServeMux (pprof already registered)
	go srv.Serve(ln)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		srv.Shutdown(ctx)
	}()

	fmt.Printf("  pprof server running at http://%s\n\n", addr)

	fmt.Println("  Endpoints registered by  import _ \"net/http/pprof\"  :")
	endpoints := []struct{ path, use string }{
		{"/debug/pprof/", "index — lists all available profiles"},
		{"/debug/pprof/profile?seconds=30", "30-second CPU profile (blocks until done)"},
		{"/debug/pprof/heap", "heap profile (live objects, inuse_space)"},
		{"/debug/pprof/allocs", "all allocations since program start"},
		{"/debug/pprof/goroutine?debug=2", "all goroutines with full stack traces"},
		{"/debug/pprof/block", "goroutines blocked on sync (needs SetBlockProfileRate)"},
		{"/debug/pprof/mutex", "contended mutex holders (needs SetMutexProfileFraction)"},
		{"/debug/pprof/trace?seconds=5", "5-second execution trace"},
		{"/debug/pprof/threadcreate", "OS thread creation stack traces"},
	}
	for _, ep := range endpoints {
		fmt.Printf("  %-50s  %s\n", "http://"+addr+ep.path, ep.use)
	}

	fmt.Println()
	fmt.Println("  Usage:")
	fmt.Printf("    go tool pprof http://%s/debug/pprof/profile?seconds=30\n", addr)
	fmt.Printf("    go tool pprof http://%s/debug/pprof/heap\n", addr)
	fmt.Printf("    go tool pprof -http=:8080 http://%s/debug/pprof/heap\n", addr) // opens browser UI
	fmt.Println()
	fmt.Println("  Typical production setup:")
	fmt.Println("    // In main() — separate port, localhost only:")
	fmt.Println("    go func() {")
	fmt.Println("        log.Println(http.ListenAndServe(\"localhost:6060\", nil))")
	fmt.Println("    }()")

	time.Sleep(100 * time.Millisecond) // keep server alive briefly for the demo
}
