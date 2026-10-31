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

Run: `go vet ./... && go test -race -v ./...`
