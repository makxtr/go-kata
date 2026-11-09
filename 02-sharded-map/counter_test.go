package shardedmap

import (
	"math/rand/v2"
	"strconv"
	"sync"
	"testing"
)

type counter interface {
	Inc(key string)
	Get(key string) int
}

type shardedCounter struct {
	m *ShardedMap[string, int]
}

func newShardedCounter(shards int) shardedCounter {
	return shardedCounter{m: New[string, int](shards)}
}

func (c shardedCounter) Inc(key string) {
	c.m.Update(key, func(old int, _ bool) int { return old + 1 })
}

func (c shardedCounter) Get(key string) int {
	v, _ := c.m.Get(key)
	return v
}

func TestCounter_Concurrent(t *testing.T) {
	tests := []struct {
		name string
		c    counter
	}{
		{"Mutex", NewCounter()},
		{"RWMutex", NewRWCounter()},
		{"SyncMap", NewSyncCounter()},
		{"Sharded", newShardedCounter(64)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var wg sync.WaitGroup

			for range 8 {
				wg.Go(func() {
					for range 1_000 {
						tt.c.Inc("hits")
					}
				})
			}
			wg.Wait()

			got := tt.c.Get("hits")
			want := 8_000

			if got != want {
				t.Fatalf("hits = %d; want %d", got, want)
			}
		})
	}
}

const keyCount = 1024

func benchKeys() []string {
	keys := make([]string, 0, keyCount)
	for i := range keyCount {
		keys = append(keys, "user"+strconv.Itoa(i))
	}
	return keys
}

func benchMixed(b *testing.B, c counter, writeEvery int) {
	keys := benchKeys()

	for _, key := range keys {
		c.Inc(key)
	}

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		i := rand.IntN(keyCount)
		n := 0

		for pb.Next() {
			key := keys[i%keyCount]
			if n%writeEvery == 0 {
				c.Inc(key)
			} else {
				c.Get(key)
			}
			i++
			n++
		}
	})
}

func BenchmarkMutex_Read95(b *testing.B) { benchMixed(b, NewCounter(), 20) }
func BenchmarkMutex_Read50(b *testing.B) { benchMixed(b, NewCounter(), 2) }

func BenchmarkRWMutex_Read95(b *testing.B) { benchMixed(b, NewRWCounter(), 20) }
func BenchmarkRWMutex_Read50(b *testing.B) { benchMixed(b, NewRWCounter(), 2) }

func BenchmarkSyncMap_Read95(b *testing.B) { benchMixed(b, NewSyncCounter(), 20) }
func BenchmarkSyncMap_Read50(b *testing.B) { benchMixed(b, NewSyncCounter(), 2) }

func BenchmarkSharded_Read95(b *testing.B) { benchMixed(b, newShardedCounter(64), 20) }
func BenchmarkSharded_Read50(b *testing.B) { benchMixed(b, newShardedCounter(64), 2) }
