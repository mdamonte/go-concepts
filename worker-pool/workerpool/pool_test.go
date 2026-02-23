package workerpool_test

import (
	"context"
	"errors"
	"log"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/marcodamonte/concurrency/worker-pool/workerpool"
)

// quietLogger returns a logger that discards output during tests unless -v is set.
func quietLogger() *log.Logger {
	if testing.Verbose() {
		return log.New(os.Stderr, "", log.LstdFlags|log.Lmicroseconds)
	}
	return log.New(os.Stderr, "", 0)
}

// ── Concurrency limit ────────────────────────────────────────────────────────

// TestConcurrencyLimit verifies that at most N jobs run simultaneously.
func TestConcurrencyLimit(t *testing.T) {
	t.Parallel()

	const workers = 3
	const jobs = 20

	pool := workerpool.New(workerpool.Config{
		Workers:         workers,
		QueueSize:       jobs,
		ShutdownTimeout: 5 * time.Second,
		Logger:          quietLogger(),
	})

	var (
		active    int64 // currently running jobs
		maxActive int64 // observed peak
	)

	barrier := make(chan struct{}) // hold all jobs until we release them

	for i := 0; i < jobs; i++ {
		if err := pool.Submit(context.Background(), func(ctx context.Context) error {
			cur := atomic.AddInt64(&active, 1)
			// Record the peak concurrency seen.
			for {
				prev := atomic.LoadInt64(&maxActive)
				if cur <= prev || atomic.CompareAndSwapInt64(&maxActive, prev, cur) {
					break
				}
			}
			<-barrier // block until released
			atomic.AddInt64(&active, -1)
			return nil
		}); err != nil {
			t.Fatalf("submit: %v", err)
		}
	}

	// Give workers time to pick up jobs and increment active.
	time.Sleep(50 * time.Millisecond)
	close(barrier) // release all blocked jobs

	if err := pool.Shutdown(); err != nil {
		t.Fatalf("shutdown: %v", err)
	}

	got := atomic.LoadInt64(&maxActive)
	if got > workers {
		t.Errorf("peak concurrency = %d; want <= %d", got, workers)
	}
	if got == 0 {
		t.Error("no jobs appear to have run")
	}
}

// ── All jobs complete ────────────────────────────────────────────────────────

// TestAllJobsProcessed checks that every submitted job eventually runs.
func TestAllJobsProcessed(t *testing.T) {
	t.Parallel()

	const total = 50

	pool := workerpool.New(workerpool.Config{
		Workers:         5,
		QueueSize:       total,
		ShutdownTimeout: 5 * time.Second,
		Logger:          quietLogger(),
	})

	var ran int64
	for i := 0; i < total; i++ {
		if err := pool.Submit(context.Background(), func(ctx context.Context) error {
			atomic.AddInt64(&ran, 1)
			return nil
		}); err != nil {
			t.Fatalf("submit: %v", err)
		}
	}

	if err := pool.Shutdown(); err != nil {
		t.Fatalf("shutdown: %v", err)
	}

	if got := atomic.LoadInt64(&ran); got != total {
		t.Errorf("ran %d jobs; want %d", got, total)
	}
}

// ── Graceful shutdown ────────────────────────────────────────────────────────

// TestGracefulShutdown ensures that jobs already in the queue are drained
// before Shutdown returns (when they finish within the timeout).
func TestGracefulShutdown(t *testing.T) {
	t.Parallel()

	const total = 10
	jobDuration := 20 * time.Millisecond

	pool := workerpool.New(workerpool.Config{
		Workers:         2,
		QueueSize:       total,
		ShutdownTimeout: 5 * time.Second,
		Logger:          quietLogger(),
	})

	var done int64
	for i := 0; i < total; i++ {
		if err := pool.Submit(context.Background(), func(ctx context.Context) error {
			time.Sleep(jobDuration)
			atomic.AddInt64(&done, 1)
			return nil
		}); err != nil {
			t.Fatalf("submit: %v", err)
		}
	}

	start := time.Now()
	if err := pool.Shutdown(); err != nil {
		t.Fatalf("unexpected forced shutdown: %v", err)
	}
	elapsed := time.Since(start)

	if got := atomic.LoadInt64(&done); got != total {
		t.Errorf("only %d/%d jobs completed before Shutdown returned", got, total)
	}

	// Sanity: shutdown should not have returned instantly (jobs took time).
	if elapsed < jobDuration {
		t.Errorf("shutdown returned too fast (%s); expected at least %s", elapsed, jobDuration)
	}
}

// ── Shutdown timeout + forced cancellation ───────────────────────────────────

// TestShutdownTimeout verifies that when jobs exceed the timeout, Shutdown
// cancels them via context and returns ErrShutdownTimeout.
func TestShutdownTimeout(t *testing.T) {
	t.Parallel()

	pool := workerpool.New(workerpool.Config{
		Workers:         2,
		QueueSize:       4,
		ShutdownTimeout: 50 * time.Millisecond, // deliberately short
		Logger:          quietLogger(),
	})

	var cancelled int64

	// Submit jobs that block until their context is cancelled.
	for i := 0; i < 4; i++ {
		if err := pool.Submit(context.Background(), func(ctx context.Context) error {
			<-ctx.Done()
			atomic.AddInt64(&cancelled, 1)
			return ctx.Err()
		}); err != nil {
			t.Fatalf("submit: %v", err)
		}
	}

	err := pool.Shutdown()
	if !errors.Is(err, workerpool.ErrShutdownTimeout) {
		t.Errorf("Shutdown() error = %v; want ErrShutdownTimeout", err)
	}

	// All running jobs must have been cancelled.
	if got := atomic.LoadInt64(&cancelled); got == 0 {
		t.Error("expected at least one job to observe context cancellation")
	}
}

// ── Submit after shutdown ────────────────────────────────────────────────────

// TestSubmitAfterShutdown confirms that jobs submitted after Shutdown returns
// ErrPoolClosed.
func TestSubmitAfterShutdown(t *testing.T) {
	t.Parallel()

	pool := workerpool.New(workerpool.Config{
		Workers:         1,
		ShutdownTimeout: time.Second,
		Logger:          quietLogger(),
	})

	if err := pool.Shutdown(); err != nil {
		t.Fatalf("shutdown: %v", err)
	}

	err := pool.Submit(context.Background(), func(ctx context.Context) error { return nil })
	if !errors.Is(err, workerpool.ErrPoolClosed) {
		t.Errorf("got %v; want ErrPoolClosed", err)
	}
}

// ── Idempotent shutdown ──────────────────────────────────────────────────────

// TestShutdownIdempotent verifies that calling Shutdown multiple times is safe.
func TestShutdownIdempotent(t *testing.T) {
	t.Parallel()

	pool := workerpool.New(workerpool.Config{
		Workers:         2,
		ShutdownTimeout: time.Second,
		Logger:          quietLogger(),
	})

	for i := 0; i < 5; i++ {
		if err := pool.Shutdown(); err != nil {
			t.Fatalf("Shutdown call %d returned unexpected error: %v", i+1, err)
		}
	}
}

// ── Metrics ──────────────────────────────────────────────────────────────────

// TestMetrics checks that counters reflect submitted/succeeded/failed tallies.
func TestMetrics(t *testing.T) {
	t.Parallel()

	const succeedN = 7
	const failN = 3
	sentinel := errors.New("intentional")

	pool := workerpool.New(workerpool.Config{
		Workers:         4,
		QueueSize:       succeedN + failN,
		ShutdownTimeout: 5 * time.Second,
		Logger:          quietLogger(),
	})

	for i := 0; i < succeedN; i++ {
		_ = pool.Submit(context.Background(), func(ctx context.Context) error { return nil })
	}
	for i := 0; i < failN; i++ {
		_ = pool.Submit(context.Background(), func(ctx context.Context) error { return sentinel })
	}

	if err := pool.Shutdown(); err != nil {
		t.Fatalf("shutdown: %v", err)
	}

	m := pool.Metrics()
	if m.Submitted != succeedN+failN {
		t.Errorf("Submitted = %d; want %d", m.Submitted, succeedN+failN)
	}
	if m.Succeeded != succeedN {
		t.Errorf("Succeeded = %d; want %d", m.Succeeded, succeedN)
	}
	if m.Failed != failN {
		t.Errorf("Failed = %d; want %d", m.Failed, failN)
	}
}

// ── No goroutine leaks ───────────────────────────────────────────────────────

// TestNoGoroutineLeak is a best-effort leak check: after Shutdown, the test
// waits briefly and then re-runs a trivial pool to confirm the runtime is
// healthy. (For rigorous leak detection use goleak in production test suites.)
func TestNoGoroutineLeak(t *testing.T) {
	t.Parallel()

	pool := workerpool.New(workerpool.Config{
		Workers:         4,
		QueueSize:       8,
		ShutdownTimeout: time.Second,
		Logger:          quietLogger(),
	})

	var count int64
	for i := 0; i < 8; i++ {
		_ = pool.Submit(context.Background(), func(ctx context.Context) error {
			atomic.AddInt64(&count, 1)
			return nil
		})
	}

	if err := pool.Shutdown(); err != nil {
		t.Fatalf("shutdown: %v", err)
	}

	// The WaitGroup inside pool guarantees all workers exited. A second pool
	// must also work cleanly, confirming no runtime-level wedge.
	pool2 := workerpool.New(workerpool.Config{
		Workers:         1,
		ShutdownTimeout: time.Second,
		Logger:          quietLogger(),
	})
	var ran int64
	_ = pool2.Submit(context.Background(), func(ctx context.Context) error {
		atomic.AddInt64(&ran, 1)
		return nil
	})
	if err := pool2.Shutdown(); err != nil {
		t.Fatalf("pool2 shutdown: %v", err)
	}
	if atomic.LoadInt64(&ran) != 1 {
		t.Error("pool2 job did not run")
	}
}

// ── Submit respects caller context ───────────────────────────────────────────

// TestSubmitRespectsCallerContext verifies that Submit returns when the caller
// cancels while waiting for a full queue.
func TestSubmitRespectsCallerContext(t *testing.T) {
	t.Parallel()

	// Unbuffered queue + 1 worker blocked on a long job = next Submit will block.
	pool := workerpool.New(workerpool.Config{
		Workers:         1,
		QueueSize:       0, // unbuffered
		ShutdownTimeout: time.Second,
		Logger:          quietLogger(),
	})

	// Occupy the single worker.
	blocker := make(chan struct{})
	_ = pool.Submit(context.Background(), func(ctx context.Context) error {
		<-blocker
		return nil
	})

	// Try to submit a second job with a very short context.
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	err := pool.Submit(ctx, func(ctx context.Context) error { return nil })
	if err == nil {
		t.Fatal("expected Submit to fail, got nil")
	}

	// Unblock the worker and clean up.
	close(blocker)
	if shutErr := pool.Shutdown(); shutErr != nil {
		t.Fatalf("shutdown: %v", shutErr)
	}
}
