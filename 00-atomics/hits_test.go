package atomics

import (
	"sync"
	"testing"
)

func TestHits_HappyPath(t *testing.T) {
	var wg sync.WaitGroup
	var h Hits
	const goroutines, perGoroutine = 8, 1_000

	for range 8 {
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
