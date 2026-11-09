package shardedmap

import "sync"

type Counter struct {
	mu sync.Mutex
	m  map[string]int
}

func NewCounter() *Counter {
	return &Counter{
		m: make(map[string]int),
	}
}

func (c *Counter) Inc(key string) {
	c.mu.Lock()
	c.m[key]++
	c.mu.Unlock()
}

func (c *Counter) Get(key string) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.m[key]
}
