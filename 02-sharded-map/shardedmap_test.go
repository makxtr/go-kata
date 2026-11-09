package shardedmap

import (
	"runtime"
	"slices"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
)

func TestShardedMap_SetGetDelete(t *testing.T) {
	m := New[string, int](8)

	if _, ok := m.Get("a"); ok {
		t.Fatalf("Get(a) on empty map: ok = true; want false")
	}

	m.Set("a", 1)
	m.Set("a", 2) // update, not a second entry

	if v, ok := m.Get("a"); !ok || v != 2 {
		t.Fatalf("Get(a) = %d, %v; want 2, true", v, ok)
	}
	if n := m.Len(); n != 1 {
		t.Fatalf("Len = %d; want 1", n)
	}

	m.Delete("a")
	m.Delete("a") // deleting a missing key is fine

	if _, ok := m.Get("a"); ok {
		t.Fatalf("Get(a) after Delete: ok = true; want false")
	}
}

func TestShardedMap_Keys(t *testing.T) {
	m := New[int, string](16)

	want := make([]int, 0, 1000)
	for i := range 1000 {
		m.Set(i, strconv.Itoa(i))
		want = append(want, i)
	}

	got := m.Keys()
	slices.Sort(got)

	if !slices.Equal(got, want) {
		t.Fatalf("Keys returned %d keys; want %d keys 0..999", len(got), len(want))
	}
}

func TestShardedMap_ShardCountBelowOne(t *testing.T) {
	for _, n := range []int{0, -5} {
		m := New[string, int](n)
		m.Set("a", 1)

		if v, ok := m.Get("a"); !ok || v != 1 {
			t.Fatalf("New(%d): Get(a) = %d, %v; want 1, true", n, v, ok)
		}
	}
}

func TestShardedMap_Distribution(t *testing.T) {
	const shards, keys = 64, 64_000

	m := New[int, struct{}](shards)
	for i := range keys {
		m.Set(i, struct{}{})
	}

	want := keys / shards
	for i := range m.shards {
		got := len(m.shards[i].m)
		if got < want/2 || got > want*2 {
			t.Errorf("shard %d holds %d keys; want about %d", i, got, want)
		}
	}
}

func TestShardedMap_UpdateIsAtomic(t *testing.T) {
	m := New[string, int](64)

	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			for range 1_000 {
				m.Update("hits", func(old int, _ bool) int { return old + 1 })
			}
		})
	}
	wg.Wait()

	if got, _ := m.Get("hits"); got != 8_000 {
		t.Fatalf("hits = %d; want 8000", got)
	}
}

func TestShardedMap_Race(t *testing.T) {
	m := New[int, int](16)

	var wg sync.WaitGroup
	for g := range 4 {
		wg.Go(func() {
			for i := range 5_000 {
				m.Set(g*10_000+i, i)
			}
		})
		wg.Go(func() {
			for i := range 5_000 {
				m.Get(g*10_000 + i)
			}
		})
		wg.Go(func() {
			for i := range 5_000 {
				m.Delete(g*10_000 + i)
			}
		})
	}
	wg.Go(func() {
		for range 50 {
			m.Keys()
		}
	})
	wg.Wait()
}

func TestShardedMap_ZeroAlloc(t *testing.T) {
	if raceEnabled {
		t.Skip("the race detector adds its own allocations")
	}

	ints := New[int, int](64)
	ints.Set(42, 1)

	strs := New[string, int](64)
	strs.Set("user42", 1)

	tests := []struct {
		name string
		fn   func()
	}{
		{"Get int", func() { ints.Get(42) }},
		{"Set int", func() { ints.Set(42, 2) }},
		{"Get string", func() { strs.Get("user42") }},
		{"Set string", func() { strs.Set("user42", 2) }},
	}

	for _, tt := range tests {
		if allocs := testing.AllocsPerRun(1000, tt.fn); allocs != 0 {
			t.Errorf("%s: %v allocs per call; want 0", tt.name, allocs)
		}
	}
}

func heapAfter[T any](fill func() T) (T, uint64) {
	var before, after runtime.MemStats

	runtime.GC()
	runtime.ReadMemStats(&before)

	v := fill()

	runtime.GC()
	runtime.ReadMemStats(&after)

	return v, after.HeapAlloc - before.HeapAlloc
}

func TestShardedMap_Memory(t *testing.T) {
	if testing.Short() {
		t.Skip("allocates about 150MB")
	}

	const n = 1_000_000
	const limit = 50 << 20

	plain, plainBytes := heapAfter(func() map[int]any {
		m := make(map[int]any)
		for i := range n {
			m[i] = i
		}
		return m
	})

	sharded, shardedBytes := heapAfter(func() *ShardedMap[int, any] {
		m := New[int, any](64)
		for i := range n {
			m.Set(i, i)
		}
		return m
	})

	extra := int64(shardedBytes) - int64(plainBytes)
	t.Logf("plain map: %d MB, sharded: %d MB, extra: %d MB",
		plainBytes>>20, shardedBytes>>20, extra>>20)

	if extra > limit {
		t.Errorf("sharded map costs %d MB more than a plain map; want under %d MB",
			extra>>20, limit>>20)
	}

	runtime.KeepAlive(plain)
	runtime.KeepAlive(sharded)
}

func BenchmarkShardedMap_Set(b *testing.B) {
	const keysPerGoroutine = 1 << 16 // bounded so the map does not grow with b.N

	for _, shards := range []int{1, 8, 64, 256} {
		b.Run("shards="+strconv.Itoa(shards), func(b *testing.B) {
			m := New[int, int](shards)

			var nextGoroutine atomic.Int64

			b.RunParallel(func(pb *testing.PB) {
				base := int(nextGoroutine.Add(1)) << 32
				i := 0

				for pb.Next() {
					m.Set(base+i%keysPerGoroutine, i)
					i++
				}
			})
		})
	}
}
