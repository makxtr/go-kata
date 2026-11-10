package atomics

import (
	"sync"
	"sync/atomic"
)

type Hits struct {
	n atomic.Int64
}

func (h *Hits) Inc() {
	h.n.Add(1)
}

func (h *Hits) Load() int64 {
	return h.n.Load()
}

type MutexHits struct {
	mu sync.Mutex
	n  int64
}

func (mh *MutexHits) Inc() {
	mh.mu.Lock()
	defer mh.mu.Unlock()
	mh.n++
}

func (mh *MutexHits) Load() int64 {
	mh.mu.Lock()
	defer mh.mu.Unlock()
	return mh.n
}
