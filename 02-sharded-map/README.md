# Kata 02: The Concurrent Map with Sharded Locks

**Target Idioms:** Concurrency Safety, Map Sharding, `sync.RWMutex`, Avoiding `sync.Map` Pitfalls
**Difficulty:** 🟡 Intermediate

## 🧠 The "Why"

Seasoned developers coming from Java might reach for `ConcurrentHashMap`-style solutions, while
Pythonistas might think of GIL-protected dictionaries. In Go, you have three main options:

- Naive `sync.Mutex` around a map (bottlenecks under high concurrency)
- `sync.Map` (optimized for specific "append-only, read-heavy" cases, but opaque and often misused)
- Sharded maps (manual control, maximized throughput)

The Go way is explicit control: if you know your access patterns, build a solution that fits.
This kata forces you to understand when and why to choose sharding over `sync.Map`.

## 🎯 The Scenario

You're building a real-time API Rate Limiter that tracks request counts per user ID. The system
handles 50k+ RPS with 95% reads (checking limits) and 5% writes (incrementing counters). A single
mutex would serialize all operations-unacceptable. `sync.Map` might work but obscures memory usage
and lacks type safety.

## 🛠 The Challenge

Implement `ShardedMap[K comparable, V any]` with configurable shard count that provides safe
concurrent access.

### 1. Functional Requirements

- Type-safe generic implementation (Go 1.18+)
- `Get(key K) (V, bool)` - returns value and existence flag
- `Set(key K, value V)` - inserts or updates
- `Delete(key K)` - removes key
- `Keys() []K` - returns all keys (order doesn't matter)
- Configurable number of shards at construction

### 2. The "Idiomatic" Constraints (Pass/Fail Criteria)

- **NO `sync.Map`:** Implement sharding manually with `[]map[K]V` and `[]sync.RWMutex`
- **Smart Sharding:** Use fnv64 hashing for key distribution (don't rely on Go's random map iteration)
- **Read Optimization:** Use `RLock()` for `Get()` operations when safe
- **Zero Allocation Hot-Path:** `Get()` and `Set()` must not allocate memory in the critical
  section (no string conversion, no boxing)
- **Clean `Keys()`:** Implement without data races, even while concurrent writes occur

## 🧪 Self-Correction (Test Yourself)

**The Contention Test**
- Run 8 goroutines doing only `Set()` operations with sequential keys
- With 1 shard: should see heavy contention (use `go test -bench=. -cpuprofile` to verify)
- With 64 shards: should see near-linear scaling

**The Memory Test**
- Store 1 million int keys with `interface{}` values
- Fail Condition: if your solution uses more than 50MB extra memory vs baseline map
- Hint: avoid `string(key)` conversions; use type-safe hashing

**The Race Test**
- Run `go test -race` with concurrent read/write/delete operations
- Any race condition = automatic failure

---

## 🗺 What is in this folder

The kata was done as a build-up: each file is one step, and each step exists because the
previous one hit a wall.

| File | Variant | What it shows |
|---|---|---|
| `counter.go` | `map` + `sync.Mutex` | The default. Correct, simple, does not scale. |
| `rwcounter.go` | `map` + `sync.RWMutex` | Readers go together. Helps only when reads dominate. |
| `synccounter.go` | `sync.Map` + `*atomic.Int64` | Lock-free reads. Needs a trick to increment. |
| `shardedmap.go` | `ShardedMap[K, V]` | The kata itself: N maps, N locks, hash picks the shard. |

Step zero, a bare map written from 8 goroutines, is not kept: it dies with
`fatal error: concurrent map writes`.

## 📊 Results

Apple M5 Pro, Go 1.27, `go test -run '^$' -bench=. -cpu=1,8`. Lower is faster. `ns/op` under
`RunParallel` is wall time divided by the operations of all goroutines together, so perfect
scaling from 1 to 8 cores would divide the number by 8.

**Mixed load on a counter, 1024 existing keys**

| Variant | 95% reads, 1 core | 95% reads, 8 cores | 50% reads, 1 core | 50% reads, 8 cores |
|---|---|---|---|---|
| `Mutex` | 8 ns | 87 ns | 9 ns | 100 ns |
| `RWMutex` | 8 ns | 36 ns | 11 ns | 71 ns |
| `sync.Map` + atomic | 13 ns | **3 ns** | 12 ns | **4 ns** |
| `ShardedMap`, 64 shards | 16 ns | 21 ns | 21 ns | 31 ns |

**The Contention Test: only `Set`, sequential int keys**

| Shards | 1 core | 8 cores |
|---|---|---|
| 1 | 17 ns | 120 ns |
| 8 | 22 ns | 77 ns |
| 64 | 22 ns | 34 ns |
| 256 | 21 ns | 28 ns |

**The Memory Test:** 1M int keys with `any` values: plain map 60 MB, sharded map 60 MB, extra 0 MB.

**CPU profile of the contention test.** With 1 shard 42% of CPU time is inside
`sync.(*RWMutex).Unlock` waking up waiters, not in the map. With 64 shards that drops to 14%.

## 📝 Notes (what I learned)

**When a map needs protection at all**

- A map needs a lock only if all three hold: more than one goroutine touches it, at least one
  of them writes, and nothing else separates them in time (`wg.Wait`, a channel, startup order).
- Scope does not matter, sharing does. A local map captured by eight closures is shared; a
  package-level map that nobody writes after `init` is safe to read from anywhere.
- Without `-race`: `fatal error: concurrent map writes`, a cheap built-in check that is not
  guaranteed to fire and cannot be recovered. With `-race`: every unsynchronized access is
  reported with both stacks.
- The lock protects the data, not the operation. `Get` must take it too: a reader walking a map
  that is growing under a writer crashes with `concurrent map read and map write`. A green test
  proves only what it exercises; the first test never read during a write.
- A nil map can be read but not written, so constructors `make` it. The zero value of
  `sync.Mutex`, `sync.RWMutex` and `sync.Map` is ready to use.

**What the numbers say**

- One lock is correct at any core count and does not scale: one goroutine works, the rest wait.
  On 8 cores it is 11 times slower than on 1, because waiting itself costs (spin, park, wake,
  cache lines moving between cores).
- That is the ceiling, not the daily reality. 87 ns/op is still 11 million ops/s; the kata's
  50k RPS keeps a single mutex busy about 0.1% of the time. "A single mutex is unacceptable"
  is an exaggeration. Measure first.
- `RWMutex` is 2.4 times faster than `Mutex` at 95% reads and 1.4 times at 50%. It still does
  not scale: every `RLock` writes to a shared reader counter, and each writer stops everyone.
  On one core it is slightly slower than `Mutex`. Default to `Mutex`, switch after a benchmark.
- Shards remove the collapse: 120 ns with 1 shard, 34 ns with 64 on the contention test. They
  do not give near-linear scaling on this machine. Every operation, even a `Get`, writes to its
  shard's lock word, and with 8 goroutines over 64 shards a few percent of operations still
  collide and pay the full price of parking. More shards help a little, with diminishing returns.
- On one core shards are twice as slow as a plain mutex: the key is hashed twice, once to pick
  the shard and once inside the map.
- `sync.Map` won this benchmark by a wide margin, and it is the only variant that got faster
  with more cores. The reason is the workload: the key set is stable, reads of existing keys
  take no lock and write nothing shared, and the increment is an atomic add on a per-key
  counter. This is exactly the "written once, read many times" case it is built for.

**sync.Map pitfalls**

- No "read, modify, write". The obvious `Load` + `Store` increment gave 1000 to 1900 instead of
  8000 in the concurrency test: lost updates. The race detector stays silent, because every
  single call is synchronized; the bug is in the gap between them. A data race and a logical
  race are different things, and `-race` finds only the first.
- The fix is to store `*atomic.Int64` and create it with `LoadOrStore`. It works, costs one
  allocation per key and gives up type safety (`v.(*atomic.Int64)`).
- Its advantage shrinks when keys are constantly added and deleted, and it cannot do an
  arbitrary update under a lock. `ShardedMap.Update` can.

**Building the sharded map**

- The kata asks for fnv64. FNV hashes bytes, and a generic `K comparable` has none without a
  conversion that allocates. `maphash.Comparable(seed, key)` (Go 1.24) hashes any comparable
  value with zero allocations; `TestShardedMap_ZeroAlloc` checks it with `testing.AllocsPerRun`.
- One slice of `struct{ mu; m }` instead of two parallel slices: the lock sits next to what it
  guards, and they cannot get out of sync.
- `Keys()` locks one shard at a time, so it is a snapshot per shard, not of the whole map.
  Locking all shards would be exact and would stop every writer. For a list of keys the
  per-shard snapshot is the honest trade-off, and it is documented on the method.
- Padding each shard to a cache line (false sharing between neighbouring locks) gave about 10%
  here. Not worth a magic number in a kata; worth knowing it exists.
- `t.Run` with a table runs one test against all four variants through a small interface
  defined in the test file, on the consumer side.

**What to use in production**

1. `map` + `sync.Mutex` by default. Simplest, any operation can run under the lock.
2. `sync.RWMutex` when a benchmark shows readers queuing and writes are a few percent.
3. `sync.Map`, or a typed library such as `xsync.Map`, for a stable key set with heavy reads.
4. A sharded map when writes are frequent, keys churn, or updates need to run under a lock,
   and a profile shows time going into lock contention. Usually from a library, not by hand.
5. A real rate limiter shared by several instances lives in Redis, not in a local map.

Run: `go vet ./... && go test -race ./... && go test -run '^$' -bench=. -cpu=1,8 ./...`
