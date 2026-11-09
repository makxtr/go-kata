package shardedmap

import (
	"strconv"
	"sync"
	"testing"
)

type counter interface {
	Inc(key string)
	Get(key string) int
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
		i := 0

		for pb.Next() {
			key := keys[i%keyCount]
			if i%writeEvery == 0 {
				c.Inc(key)
			} else {
				c.Get(key)
			}
			i++
		}
	})
}

func BenchmarkMutex_Read95(b *testing.B) { benchMixed(b, NewCounter(), 20) }
func BenchmarkMutex_Read50(b *testing.B) { benchMixed(b, NewCounter(), 2) }

func TestCounter_Concurrent(t *testing.T) {
	var wg sync.WaitGroup

	c := NewCounter()

	for range 8 {
		wg.Go(func() {
			for range 1_000 {
				c.Inc("hits")
			}
		})
	}
	wg.Wait()

	got := c.Get("hits")
	want := 8_000

	if got != want {
		t.Fatalf("hits = %d; want %d", got, want)
	}
}
