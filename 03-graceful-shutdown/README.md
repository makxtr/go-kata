# Kata 03: The Graceful Shutdown Server

**Target Idioms:** Context Propagation, Signal Handling, Channel Coordination, Resource Cleanup
**Difficulty:** 🔴 Advanced

## 🧠 The "Why"

In other ecosystems, graceful shutdown is often framework magic (Spring's `@PreDestroy`,
Django's `close()`). Go forces explicit lifecycle management. The mismatch: developers used to
automatic cleanup often leak goroutines, drop in-flight requests, or corrupt data during shutdown.

The Go way: **own your lifecycle.** Every goroutine you spawn must have a controlled shutdown path.

## 🎯 The Scenario

Build an HTTP Server with Background Worker that must:

- Accept HTTP requests (handled by a pool of worker goroutines)
- Run a background cache warmer every 30 seconds
- Maintain persistent database connections
- Shutdown within 10 seconds when receiving SIGTERM, completing in-flight requests but
  rejecting new ones

## 🛠 The Challenge

Implement a `Server` struct with `Start() error` and `Stop(ctx context.Context) error` methods.

### 1. Functional Requirements

- HTTP server on configurable port with request timeout
- Worker pool (configurable size) processes requests via channel
- Background cache warmer ticks every 30s (use `time.Ticker`)
- Database connection pool (mock with `net.Conn`)
- SIGTERM/SIGINT triggers graceful shutdown
- Shutdown completes within deadline or forces exit

### 2. The "Idiomatic" Constraints (Pass/Fail Criteria)

- **Single Context Tree:** root `context.Context` passed to `Start()`, canceled on shutdown
- **Channel Coordination:** use `chan struct{}` for worker pool shutdown, not boolean flags
- **Proper Ticker Cleanup:** `defer ticker.Stop()` with `select` in goroutine
- **Dependency Order:** shutdown in reverse order
  (stop accepting → drain workers → stop warmer → close DB)
- **No `os.Exit()` in business logic:** shutdown should be testable without process termination

## 🧪 Self-Correction (Test Yourself)

**The Sudden Death Test**
- Send 100 requests, immediately send SIGTERM
- Pass: server completes in-flight requests (not all 100), logs "shutting down", closes cleanly
- Fail: server accepts new requests after signal, leaks goroutines, or crashes

**The Slow Leak Test**
- Run server for 5 minutes with 1 request/second
- Send SIGTERM, wait 15 seconds
- Pass: `go test` shows no goroutine leaks (use `runtime.NumGoroutine()`)
- Fail: any increase in goroutine count from start to finish

**The Timeout Test**
- Start long-running request (sleep 20s)
- Send SIGTERM with 5s timeout context
- Pass: forces shutdown after 5s, logs "shutdown timeout"
- Fail: waits full 20s or deadlocks

## 📚 Resources

- [Go Blog: Context](https://go.dev/blog/context)
- [Graceful Shutdown in Go](https://pkg.go.dev/net/http#Server.Shutdown)
- [Signal Handling](https://pkg.go.dev/os/signal)

---

## 📝 Notes (what I learned)

- `":0"` as the port lets the OS pick a free one; use `net.Listen` + `Serve(listener)`
  instead of `ListenAndServe` so the chosen address can be read back.
- Every time constant in a service must be an option, otherwise tests cannot shrink it.
- `Start` is non-blocking: it listens first (so nothing leaks on error), then spawns
  workers, warmer and `Serve` in goroutines and returns. `Stop` does the waiting.
- Stop is a chain of "signal, then wait" pairs in reverse start order:
  `Shutdown` + `<-serveErr`, `close(quit)` + `wg.Wait()`, `close(warmQuit)` + `<-warmDone`,
  then `db.Close()`. Closing a `chan struct{}` is a broadcast: every goroutine selecting on it
  wakes up at once.
- `WaitGroup` for N identical goroutines, a `done` channel for a single one, `chan error`
  with buffer 1 for a goroutine that returns a result (`Serve`).
- `http.Server.Shutdown` closes the listener and idle connections, then waits for active
  handlers. It does NOT cancel `r.Context()`, so in-flight requests finish. The ctx passed to
  `Shutdown` is only an upper bound. On `DeadlineExceeded` fall back to `Close()`, which drops
  connections and does cancel `r.Context()`.
- A worker must never block uncancellably: `select` on `time.After` and `quit` instead of
  `time.Sleep`, and `Job.Result` has buffer 1 so a worker whose handler already timed out
  can drop the result and move on (without the buffer the pool silently shrinks and `Stop`
  deadlocks on `wg.Wait`).
- Signals belong in `main`: `signal.NotifyContext` turns SIGINT/SIGTERM into ctx cancellation.
  The shutdown ctx must derive from `context.Background()`, the signal ctx is already cancelled.
  Call `stop()` after the first signal so a second Ctrl+C kills the process the default way.
- `os.Exit` skips deferred calls; keep it out of library code so shutdown stays testable.
- Goroutine-leak test: `runtime.NumGoroutine()` before and after, but goroutines exit
  asynchronously, so poll with a deadline instead of comparing once. The HTTP client keeps
  keep-alive connections with two goroutines each: call
  `http.DefaultTransport.(*http.Transport).CloseIdleConnections()` before counting.
- `t.Fatal` only from the test goroutine; from spawned goroutines use `t.Error`.
- `wg.Go` (Go 1.25) replaces the `Add`/`go`/`defer Done` triple.
- Sudden Death clients fail in three ways: `connection refused` (listener closed),
  `connection reset` (accepted by the kernel, never by `Accept`), `EOF` (idle conn closed).
  One client may take a full second: a lost SYN is retransmitted after the TCP timeout.

Run: `go vet ./... && go test -race -v ./...`
