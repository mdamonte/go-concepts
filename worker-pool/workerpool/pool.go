// Package workerpool provides a fixed-size worker pool with graceful shutdown,
// context-based cancellation, and basic observability via atomic counters and
// structured logging.
package workerpool

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

// Job is the unit of work submitted to the pool. The function receives the
// pool's context so it can respect cancellation.
type Job func(ctx context.Context) error

// Config holds pool construction parameters.
type Config struct {
	// Workers is the number of goroutines that consume jobs concurrently.
	Workers int

	// QueueSize is the capacity of the internal job channel. A value of 0
	// makes the channel unbuffered (submit blocks until a worker is free).
	QueueSize int

	// ShutdownTimeout is the maximum time Shutdown waits for in-flight jobs
	// to finish before forcefully cancelling them. Defaults to 30 s.
	ShutdownTimeout time.Duration

	// Logger is used for structured output. If nil, log.Default() is used.
	Logger *log.Logger
}

func (c *Config) withDefaults() Config {
	out := *c
	if out.Workers <= 0 {
		out.Workers = 1
	}
	if out.ShutdownTimeout <= 0 {
		out.ShutdownTimeout = 30 * time.Second
	}
	if out.Logger == nil {
		out.Logger = log.Default()
	}
	return out
}

// Metrics exposes live pool counters. All fields are updated atomically and
// safe to read from any goroutine.
type Metrics struct {
	Submitted int64 // total jobs ever enqueued
	Started   int64 // jobs a worker picked up
	Succeeded int64 // jobs that returned nil
	Failed    int64 // jobs that returned a non-nil error
	Dropped   int64 // jobs rejected after shutdown began
}

// Pool is a fixed-size worker pool.
//
// Lifecycle:
//
//	pool := workerpool.New(cfg)
//	pool.Submit(job)      // non-blocking if queue has space
//	pool.Shutdown()       // stop accepting, drain, cancel stragglers
type Pool struct {
	cfg     Config
	jobs    chan Job
	wg      sync.WaitGroup // tracks live worker goroutines
	metrics Metrics

	// cancelWorkers stops workers when ShutdownTimeout elapses.
	cancelWorkers context.CancelFunc
	workerCtx     context.Context

	// once ensures Shutdown is idempotent.
	once sync.Once

	// closed is set to 1 atomically when Shutdown begins; Submit reads it.
	closed int32
}

// New creates a Pool and starts N worker goroutines. Workers run until
// Shutdown is called.
func New(cfg Config) *Pool {
	cfg = cfg.withDefaults()

	workerCtx, cancelWorkers := context.WithCancel(context.Background())

	p := &Pool{
		cfg:           cfg,
		jobs:          make(chan Job, cfg.QueueSize),
		workerCtx:     workerCtx,
		cancelWorkers: cancelWorkers,
	}

	p.cfg.Logger.Printf("[pool] starting %d workers (queue=%d, shutdownTimeout=%s)",
		cfg.Workers, cfg.QueueSize, cfg.ShutdownTimeout)

	for i := 0; i < cfg.Workers; i++ {
		p.wg.Add(1)
		go p.runWorker(i)
	}

	return p
}

// Submit enqueues a job. It returns ErrPoolClosed if the pool is shutting down,
// or ErrQueueFull if the internal channel is full (only possible with a buffered
// queue and a non-blocking send path — here we block on send).
//
// Submit blocks if the queue is full, respecting the caller's context so
// the caller can time-out or cancel the submission itself.
func (p *Pool) Submit(ctx context.Context, job Job) error {
	if atomic.LoadInt32(&p.closed) == 1 {
		atomic.AddInt64(&p.metrics.Dropped, 1)
		return ErrPoolClosed
	}

	atomic.AddInt64(&p.metrics.Submitted, 1)

	select {
	case p.jobs <- job:
		return nil
	case <-ctx.Done():
		// Caller cancelled while waiting for queue space.
		atomic.AddInt64(&p.metrics.Dropped, 1)
		return fmt.Errorf("submit cancelled: %w", ctx.Err())
	}
}

// Shutdown stops the pool gracefully:
//  1. Marks the pool as closed so no new jobs are accepted.
//  2. Closes the jobs channel so workers drain the remaining queue and exit.
//  3. Waits up to ShutdownTimeout for workers to finish.
//  4. If the timeout elapses, cancels all worker contexts and waits for
//     workers to exit (they must respect ctx cancellation).
//
// Shutdown is safe to call more than once; subsequent calls are no-ops.
// It returns ErrShutdownTimeout if a forced cancellation was required.
func (p *Pool) Shutdown() error {
	var shutdownErr error

	p.once.Do(func() {
		p.cfg.Logger.Printf("[pool] shutdown initiated")

		// 1. Stop accepting new jobs.
		atomic.StoreInt32(&p.closed, 1)

		// 2. Signal workers: no more jobs will arrive.
		close(p.jobs)

		// 3. Wait up to ShutdownTimeout for a clean drain.
		done := make(chan struct{})
		go func() {
			p.wg.Wait()
			close(done)
		}()

		select {
		case <-done:
			p.cfg.Logger.Printf("[pool] shutdown complete (all workers exited cleanly)")

		case <-time.After(p.cfg.ShutdownTimeout):
			// 4. Timeout: force-cancel in-flight jobs.
			p.cfg.Logger.Printf("[pool] shutdown timeout (%s) elapsed — cancelling workers",
				p.cfg.ShutdownTimeout)
			p.cancelWorkers()
			<-done // wait for workers to ack cancellation
			p.cfg.Logger.Printf("[pool] shutdown complete (forced)")
			shutdownErr = ErrShutdownTimeout
		}
	})

	return shutdownErr
}

// Metrics returns a snapshot of pool counters. Values are consistent within
// each field but may not be mutually consistent across fields (no global lock).
func (p *Pool) Metrics() Metrics {
	return Metrics{
		Submitted: atomic.LoadInt64(&p.metrics.Submitted),
		Started:   atomic.LoadInt64(&p.metrics.Started),
		Succeeded: atomic.LoadInt64(&p.metrics.Succeeded),
		Failed:    atomic.LoadInt64(&p.metrics.Failed),
		Dropped:   atomic.LoadInt64(&p.metrics.Dropped),
	}
}

// runWorker is the goroutine body for one worker.
func (p *Pool) runWorker(id int) {
	defer p.wg.Done()
	p.cfg.Logger.Printf("[worker %d] started", id)

	for job := range p.jobs {
		// Check whether a force-cancel happened before we even start.
		if p.workerCtx.Err() != nil {
			p.cfg.Logger.Printf("[worker %d] skipping job: context already cancelled", id)
			atomic.AddInt64(&p.metrics.Failed, 1)
			continue
		}

		atomic.AddInt64(&p.metrics.Started, 1)

		if err := job(p.workerCtx); err != nil {
			atomic.AddInt64(&p.metrics.Failed, 1)
			p.cfg.Logger.Printf("[worker %d] job failed: %v", id, err)
		} else {
			atomic.AddInt64(&p.metrics.Succeeded, 1)
		}
	}

	p.cfg.Logger.Printf("[worker %d] exited", id)
}

// Sentinel errors returned by the pool.
var (
	ErrPoolClosed      = fmt.Errorf("worker pool is closed")
	ErrShutdownTimeout = fmt.Errorf("shutdown timeout elapsed; workers were force-cancelled")
)
