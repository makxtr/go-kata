# Kata 01: The Fail-Fast Data Aggregator

**Target Idioms:** Concurrency Control (errgroup), Context Propagation, Functional Options
**Difficulty:** 🟡 Intermediate

## 🧠 The "Why"

In other languages, you might use `Promise.all` or strict thread pools to fetch data in parallel.
In Go, seasoned developers often start with `sync.WaitGroup`, but quickly realize it lacks two
critical features for production: **Error Propagation** and **Context Cancellation**.

If you spawn 10 goroutines and the first one fails, `WaitGroup` blindly waits for the other 9
to finish. Idiomatic Go fails fast.

## 🎯 The Scenario

You are building a User Dashboard Backend. To render the dashboard, you must fetch data from two
independent, mock microservices:

- Profile Service (returns `"Name: Alice"`)
- Order Service (returns `"Orders: 5"`)

You need to fetch these in parallel to reduce latency. However, if either fails, or if the global
timeout is reached, the entire operation must abort immediately to save resources.

## 🛠 The Challenge

Create a `UserAggregator` struct and a method `Aggregate(id int)` that orchestrates this fetching.

### 1. Functional Requirements

- The aggregator must be configurable (timeout, logger) without a massive constructor.
- Both services must be queried concurrently.
- The result should combine both outputs: `"User: Alice | Orders: 5"`.

### 2. The "Idiomatic" Constraints (Pass/Fail Criteria)

- **NO `sync.WaitGroup`:** you must use `golang.org/x/sync/errgroup`.
- **NO "Parameter Soup":** you must use the Functional Options Pattern for the constructor
  (e.g., `New(WithTimeout(2s))`).
- **Context is King:** you must pass `context.Context` as the first argument to your methods.
- **Cleanup:** if the Profile service fails, the Order service request must be cancelled
  (via Context) immediately.
- **Modern Logging:** use `log/slog` for structured logging.

## 🧪 Self-Correction (Test Yourself)

**The "Slow Poke"**
- Set your aggregator timeout to 1s.
- Mock one service to take 2s.
- Pass condition: does your function return `context deadline exceeded` after exactly 1s?

**The "Domino Effect"**
- Mock the Profile Service to return an error immediately.
- Mock the Order Service to take 10 seconds.
- Pass condition: does your function return the error immediately?
  (If it waits 10s, you failed context cancellation.)

## 📚 Resources

- [Go Concurrency: errgroup](https://pkg.go.dev/golang.org/x/sync/errgroup)
- [Functional Options for Friendly APIs](https://dave.cheney.net/2014/10/17/functional-options-for-friendly-apis)

---

## 📝 Notes (what I learned)

- `errgroup.WithContext` = WaitGroup + first error + context cancellation.
- Cancellation is cooperative: errgroup closes `ctx.Done()`, but the fetcher must listen
  to it (`select` on `ctx.Done()`); a `time.Sleep` ignores it and `Wait` keeps waiting.
- `context.DeadlineExceeded` comes from the timer, `context.Canceled` from a failed sibling.
- Wrap errors with `%w` so `errors.Is` can see through the chain.
- `context.Background()` in the middle of a call chain breaks the cancellation tree.
- A test that never failed proves nothing: make it red first, then green.

Run: `go vet ./... && go test -race -v ./...`
