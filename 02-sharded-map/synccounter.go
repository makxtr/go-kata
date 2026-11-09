package shardedmap

import (
	"sync"
	"sync/atomic"
)

type SyncCounter struct {
	m sync.Map
}

func NewSyncCounter() *SyncCounter {
	return &SyncCounter{}
}

func (c *SyncCounter) Inc(key string) {
	v, ok := c.m.Load(key)
	if !ok {
		v, _ = c.m.LoadOrStore(key, new(atomic.Int64))
	}
	v.(*atomic.Int64).Add(1)
}

func (c *SyncCounter) Get(key string) int {
	v, ok := c.m.Load(key)
	if !ok {
		return 0
	}
	return int(v.(*atomic.Int64).Load())
}
