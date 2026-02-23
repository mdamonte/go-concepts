package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/marcodamonte/concurrency/worker-pool/workerpool"
)

func main() {
	logger := log.New(os.Stdout, "", log.LstdFlags|log.Lmicroseconds)

	pool := workerpool.New(workerpool.Config{
		Workers:         4,
		QueueSize:       20,
		ShutdownTimeout: 3 * time.Second,
		Logger:          logger,
	})

	// ── Graceful shutdown on SIGINT / SIGTERM ────────────────────────────────
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// ── Submit jobs until the signal fires ──────────────────────────────────
	go func() {
		for id := 1; ; id++ {
			select {
			case <-ctx.Done():
				return
			default:
			}

			jobID := id // capture for closure
			err := pool.Submit(ctx, func(jobCtx context.Context) error {
				return processOrder(jobCtx, jobID)
			})

			switch {
			case errors.Is(err, workerpool.ErrPoolClosed):
				return
			case err != nil:
				// Submit was cancelled because the signal fired mid-wait.
				return
			}

			// Pace submissions so the demo is readable.
			select {
			case <-time.After(80 * time.Millisecond):
			case <-ctx.Done():
			}
		}
	}()

	// ── Wait for signal ──────────────────────────────────────────────────────
	<-ctx.Done()
	stop() // release signal resources

	fmt.Println()
	logger.Println("[main] signal received — shutting down pool")

	if err := pool.Shutdown(); errors.Is(err, workerpool.ErrShutdownTimeout) {
		logger.Println("[main] some jobs were cancelled (shutdown timeout exceeded)")
	}

	m := pool.Metrics()
	logger.Printf("[main] metrics: submitted=%d started=%d succeeded=%d failed=%d dropped=%d",
		m.Submitted, m.Started, m.Succeeded, m.Failed, m.Dropped)
}

// processOrder simulates order processing with variable latency and occasional
// failures. It respects ctx so it can be cancelled during a forced shutdown.
func processOrder(ctx context.Context, id int) error {
	// Simulate variable work duration (100–500 ms).
	duration := time.Duration(100+rand.Intn(400)) * time.Millisecond

	log.Printf("[job %3d] started  (will take %s)", id, duration.Round(time.Millisecond))

	select {
	case <-time.After(duration):
	case <-ctx.Done():
		log.Printf("[job %3d] cancelled: %v", id, ctx.Err())
		return ctx.Err()
	}

	// Simulate ~10 % failure rate.
	if rand.Intn(10) == 0 {
		err := fmt.Errorf("payment gateway timeout for order %d", id)
		log.Printf("[job %3d] failed:    %v", id, err)
		return err
	}

	log.Printf("[job %3d] done", id)
	return nil
}
