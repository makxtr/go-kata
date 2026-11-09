package shardedmap

import "sync"

type RWCounter struct {
	mu sync.RWMutex
	m  map[string]int
}

func NewRWCounter() *RWCounter {
	return &RWCounter{
		m: make(map[string]int),
	}
}

func (c *RWCounter) Inc(key string) {
	c.mu.Lock()
	c.m[key]++
	c.mu.Unlock()
}

func (c *RWCounter) Get(key string) int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.m[key]
}
