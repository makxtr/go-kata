package atomics

import (
	"sync"
	"testing"
)

func TestHits_HappyPath(t *testing.T) {
	var wg sync.WaitGroup
	var h Hits
	const goroutines, perGoroutine = 8, 1_000

	for range goroutines {
		wg.Go(func() {
			for range perGoroutine {
				h.Inc()
			}
		})
	}

	wg.Wait()

	if got := h.Load(); got != goroutines*perGoroutine {
		t.Fatalf("got %d, want %d", got, goroutines*perGoroutine)
	}
}

func BenchmarkHits_Inc(b *testing.B) {
	var h Hits
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			h.Inc()
		}
	})
}

func BenchmarkMutexHits_Inc(b *testing.B) {
	var mh MutexHits
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			mh.Inc()
		}
	})
}
