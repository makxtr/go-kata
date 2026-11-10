package atomics

import "sync/atomic"

type Hits struct {
	n atomic.Int64
}

func (h *Hits) Inc() {
	h.n.Add(1)
}

func (h *Hits) Load() int64 {
	return h.n.Load()
}
