// Package textcache holds what a backend's text engine has shaped for a line, so a
// width, a caret, a click and a draw of the same line shape it once.
//
// A hit costs a map lookup and allocates nothing. Entries age out by generation: when
// the current generation is full it becomes the previous one, and the previous one is
// released; a line used again while it is in the previous generation moves back. A
// line painted every frame therefore never leaves, and a line that stops being
// painted is released within two generations. Not safe for concurrent use: it lives
// on the paint goroutine with the engine that owns it.
package textcache

import "strings"

// Key names one shaped line.
type Key struct {
	Value        string
	Size, Weight float32
}

// Cache maps keys to a platform's shaped lines.
type Cache[V any] struct {
	current, previous map[Key]V
	limit             int
	release           func(V)
}

// New returns a cache that keeps up to 2*limit entries. release, when not nil, is
// called once for each value the cache drops, so a native handle is freed with it.
func New[V any](limit int, release func(V)) *Cache[V] {
	limit = max(limit, 1)
	return &Cache[V]{current: make(map[Key]V, limit), previous: make(map[Key]V, limit), limit: limit, release: release}
}

// Get is the value stored for key.
func (cache *Cache[V]) Get(key Key) (V, bool) {
	if value, ok := cache.current[key]; ok {
		return value, true
	}
	value, ok := cache.previous[key]
	if ok {
		delete(cache.previous, key)
		cache.insert(key, value)
	}
	return value, ok
}

// Put stores value for key. The key's string is copied, so the caller may pass text
// that aliases a buffer it reuses. A value already stored for key is released.
func (cache *Cache[V]) Put(key Key, value V) {
	if old, ok := cache.current[key]; ok {
		cache.drop(old)
		cache.current[key] = value
		return
	}
	if old, ok := cache.previous[key]; ok {
		delete(cache.previous, key)
		cache.drop(old)
	}
	key.Value = strings.Clone(key.Value)
	cache.insert(key, value)
}

func (cache *Cache[V]) insert(key Key, value V) {
	if len(cache.current) >= cache.limit {
		cache.rotate()
	}
	cache.current[key] = value
}

// rotate releases the previous generation and makes the current one previous. The
// emptied map is reused, so a steady working set does not allocate.
func (cache *Cache[V]) rotate() {
	for _, value := range cache.previous {
		cache.drop(value)
	}
	clear(cache.previous)
	cache.current, cache.previous = cache.previous, cache.current
}

func (cache *Cache[V]) drop(value V) {
	if cache.release != nil {
		cache.release(value)
	}
}

// Clear releases every entry: the font set or the display scale changed and no
// shaped line is valid.
func (cache *Cache[V]) Clear() {
	for _, value := range cache.current {
		cache.drop(value)
	}
	for _, value := range cache.previous {
		cache.drop(value)
	}
	clear(cache.current)
	clear(cache.previous)
}

// Len is the number of entries held.
func (cache *Cache[V]) Len() int { return len(cache.current) + len(cache.previous) }
