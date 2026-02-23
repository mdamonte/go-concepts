package main

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"time"

	_ "net/http/pprof"
)

type result struct {
	Service string
	Value   string
	Err     error
	Latency time.Duration
}

func main() {
	rand.Seed(time.Now().UnixNano())
	go func() {
		http.ListenAndServe("localhost:6060", nil)
	}()

	// Cancelación manual (Ctrl+C)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	// Timeout global de la request
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	resultsCh := make(chan result)

	// Lanzamos 2 "llamadas" concurrentes
	go callService(ctx, "payments", 3*time.Millisecond, 6*time.Millisecond, resultsCh)
	go callService(ctx, "shipping", 3*time.Millisecond, 6*time.Millisecond, resultsCh)

	// Recolectamos 2 resultados o cancelamos
	want := 2
	got := 0
	var results []result

	for got < want {
		select {
		case r := <-resultsCh:
			got++
			results = append(results, r)
			if r.Err != nil {
				fmt.Printf("❌ %s failed after %s: %v\n", r.Service, r.Latency, r.Err)
			} else {
				fmt.Printf("✅ %s ok after %s: %s\n", r.Service, r.Latency, r.Value)
			}
		case <-ctx.Done():
			// Si el contexto se canceló (timeout o Ctrl+C), cortamos ordenadamente.
			fmt.Printf("\n🛑 stopped: %v\n", ctx.Err())
			printSummary(results, want)
			return
		}
	}

	fmt.Println("\n🎉 all services finished")
	printSummary(results, want)
}

func callService(ctx context.Context, name string, minDelay, maxDelay time.Duration, out chan<- result) {
	// Simulamos latencia variable
	delay := minDelay + time.Duration(rand.Int63n(int64(maxDelay-minDelay)))
	time.Sleep(5 * time.Second)
	start := time.Now()
	select {
	case <-time.After(delay):
		// “Terminó” la llamada
		out <- result{
			Service: name,
			Value:   fmt.Sprintf("%s-response", name),
			Err:     nil,
			Latency: time.Since(start),
		}
	case <-ctx.Done():
		// Se canceló antes de terminar: salimos sin colgar goroutines
		out <- result{
			Service: name,
			Value:   "",
			Err:     ctx.Err(),
			Latency: time.Since(start),
		}
	}
}

func printSummary(results []result, want int) {
	fmt.Printf("\n--- summary (%d/%d collected) ---\n", len(results), want)
	for _, r := range results {
		if r.Err != nil {
			fmt.Printf("- %s: err=%v (after %s)\n", r.Service, r.Err, r.Latency)
		} else {
			fmt.Printf("- %s: ok=%s (after %s)\n", r.Service, r.Value, r.Latency)
		}
	}
}
