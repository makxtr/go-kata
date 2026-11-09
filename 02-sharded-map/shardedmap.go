package shardedmap

import (
	"hash/maphash"
	"sync"
)

type shard[K comparable, V any] struct {
	mu sync.RWMutex
	m  map[K]V
}

type ShardedMap[K comparable, V any] struct {
	shards []shard[K, V]
	seed   maphash.Seed
}

func New[K comparable, V any](shardCount int) *ShardedMap[K, V] {
	if shardCount < 1 {
		shardCount = 1
	}

	shards := make([]shard[K, V], shardCount)
	for i := range shards {
		shards[i].m = make(map[K]V)
	}

	return &ShardedMap[K, V]{
		shards: shards,
		seed:   maphash.MakeSeed(),
	}
}

func (sm *ShardedMap[K, V]) shardFor(key K) *shard[K, V] {
	h := maphash.Comparable(sm.seed, key)
	return &sm.shards[h%uint64(len(sm.shards))]
}

func (sm *ShardedMap[K, V]) Get(key K) (V, bool) {
	s := sm.shardFor(key)

	s.mu.RLock()
	defer s.mu.RUnlock()

	v, ok := s.m[key]
	return v, ok
}

func (sm *ShardedMap[K, V]) Set(key K, value V) {
	s := sm.shardFor(key)

	s.mu.Lock()
	defer s.mu.Unlock()

	s.m[key] = value
}

func (sm *ShardedMap[K, V]) Delete(key K) {
	s := sm.shardFor(key)

	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.m, key)
}

func (sm *ShardedMap[K, V]) Update(key K, fn func(old V, existed bool) V) {
	s := sm.shardFor(key)

	s.mu.Lock()
	defer s.mu.Unlock()

	old, ok := s.m[key]
	s.m[key] = fn(old, ok)
}

func (sm *ShardedMap[K, V]) Keys() []K {
	keys := make([]K, 0, sm.Len())

	for i := range sm.shards {
		s := &sm.shards[i]

		s.mu.RLock()
		for k := range s.m {
			keys = append(keys, k)
		}
		s.mu.RUnlock()
	}

	return keys
}

func (sm *ShardedMap[K, V]) Len() int {
	n := 0

	for i := range sm.shards {
		s := &sm.shards[i]

		s.mu.RLock()
		n += len(s.m)
		s.mu.RUnlock()
	}

	return n
}
