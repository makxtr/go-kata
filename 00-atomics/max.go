package atomics

import "sync/atomic"

type Max struct {
	n atomic.Int64
}

func (m *Max) Observe(v int64) {
	for {
		cur := m.n.Load()
		if v <= cur {
			return
		}
		if m.n.CompareAndSwap(cur, v) {
			return
		}
	}
}

func (m *Max) Load() int64 {
	return m.n.Load()
}
