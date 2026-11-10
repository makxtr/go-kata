package atomics

import (
	"sync"
	"testing"
)

func TestMax_Observe(t *testing.T) {
	var wg sync.WaitGroup
	var m Max
	const goroutines, perGoroutine = 8, 1_000

	for i := range goroutines {
		wg.Go(func() {
			base := int64(i * perGoroutine)
			for j := perGoroutine - 1; j >= 0; j-- {
				m.Observe(base + int64(j))
			}
		})
	}

	wg.Wait()

	const want = goroutines*perGoroutine - 1
	if got := m.Load(); got != want {
		t.Fatalf("got %d, want %d", got, want)
	}
}

func TestMax_ZeroValue(t *testing.T) {
	var m Max
	if got := m.Load(); got != 0 {
		t.Fatalf("got %d, want 0", got)
	}

	m.Observe(-5)
	if got := m.Load(); got != 0 {
		t.Fatalf("after Observe(-5): got %d, want 0", got)
	}
}
