# Worker Pool

A production-quality, fixed-size worker pool in Go with graceful shutdown,
context propagation, and observability basics.

---

## Project layout

```
worker-pool/
├── go.mod
├── main.go                  # runnable demo (order-processing simulation)
└── workerpool/
    ├── pool.go              # pool implementation
    └── pool_test.go         # unit tests
```

---

## Design

### Core types

| Type | Role |
|------|------|
| `Pool` | Owns the job channel, worker goroutines, and shutdown logic |
| `Job` | `func(ctx context.Context) error` – the unit of work |
| `Config` | Workers, QueueSize, ShutdownTimeout, Logger |
| `Metrics` | Atomic counters: Submitted / Started / Succeeded / Failed / Dropped |

### Channel topology

```
Submit()  ──►  jobs chan Job  ──►  worker-0
                               ──►  worker-1
                               ──►  ...
                               ──►  worker-N
```

- `jobs` is a single shared channel; Go's scheduler distributes work fairly
  across workers without any explicit synchronisation.
- `QueueSize = 0` → unbuffered; `Submit` blocks until a worker is free.
- `QueueSize > 0` → buffered; `Submit` only blocks when the buffer is full.

### Context layers

```
context.Background()
    └── workerCtx  (cancelled only on forced shutdown timeout)
            └── passed to every Job as the first argument
```

Callers of `Submit` pass their **own** context, which governs how long they
are willing to wait for queue space; it does **not** cancel in-flight jobs.

---

## Shutdown flow

```
pool.Shutdown()
    │
    ├─ 1. atomic.StoreInt32(&closed, 1)   → Submit() returns ErrPoolClosed
    │
    ├─ 2. close(jobs)                     → workers' range loop exits after
    │                                        draining remaining items
    │
    ├─ 3. wait for wg.Wait() with a timer
    │       │
    │       ├─ wg.Wait() fires first  → clean shutdown ✓
    │       │
    │       └─ timeout fires first
    │               │
    │               ├─ cancelWorkers()  → workerCtx.Done() is closed;
    │               │                    jobs select on ctx.Done() and return
    │               │
    │               └─ wait for wg.Wait() → forced shutdown, returns
    │                                        ErrShutdownTimeout
    │
    └─ sync.Once ensures all of the above runs exactly once
```

**Guarantee**: worker goroutines always reach `wg.Done()` — no leaks.

---

## Observability

Metrics are updated with `sync/atomic` and can be read at any time from any
goroutine without a lock:

```go
m := pool.Metrics()
fmt.Printf("submitted=%d started=%d succeeded=%d failed=%d dropped=%d",
    m.Submitted, m.Started, m.Succeeded, m.Failed, m.Dropped)
```

Structured log lines (compatible with any `*log.Logger`):

```
[pool]     starting 4 workers (queue=20, shutdownTimeout=3s)
[worker 0] started
[worker 1] started
[job   1]  started  (will take 213ms)
[worker 0] exited
[pool]     shutdown complete (all workers exited cleanly)
```

---

## Running the demo

```bash
cd worker-pool
go run .
```

Press **Ctrl-C** to trigger graceful shutdown. The pool will:
1. Stop accepting new orders.
2. Wait up to 3 s for in-flight orders to complete.
3. Force-cancel any remaining orders and exit.

---

## Running the tests

```bash
# All tests, with race detector
go test -race ./workerpool/...

# Verbose output
go test -race -v ./workerpool/...

# A single test
go test -race -run TestShutdownTimeout ./workerpool/...
```

### Test coverage

| Test | What it verifies |
|------|-----------------|
| `TestConcurrencyLimit` | Peak concurrency ≤ N workers |
| `TestAllJobsProcessed` | Every submitted job runs exactly once |
| `TestGracefulShutdown` | Queue drains cleanly within timeout |
| `TestShutdownTimeout` | Forced cancel returns `ErrShutdownTimeout` |
| `TestSubmitAfterShutdown` | Returns `ErrPoolClosed` |
| `TestShutdownIdempotent` | Multiple `Shutdown()` calls are safe |
| `TestMetrics` | Counters match submitted/succeeded/failed counts |
| `TestNoGoroutineLeak` | A second pool works after first shuts down |
| `TestSubmitRespectsCallerContext` | Blocked `Submit` respects caller cancellation |

---

## Trade-offs & extension points

| Decision | Rationale | Alternative |
|----------|-----------|-------------|
| Single shared channel | Simple, fair, low overhead | Per-worker queues for locality |
| `sync/atomic` counters | No lock contention on hot path | `expvar` or Prometheus gauge |
| `*log.Logger` for logging | Zero dependencies | `slog` (Go 1.21+), `zap`, `zerolog` |
| `sync.Once` for shutdown | Idempotent, race-free | `chan struct{}` with `select` |
| `wg.Wait` in goroutine | Allows `select` with timer | `time.AfterFunc` |
