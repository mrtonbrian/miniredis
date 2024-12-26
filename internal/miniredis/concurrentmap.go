package miniredis

import "sync"

// A simple concurrent map with using a RWMutex.
// Probably will switch it out in the future, or implement sharding
type ConcurrentMap[K comparable, T any] struct {
	Map   map[K]T
	Mutex sync.RWMutex
}

// NewConcurrentMap creates a new ConcurrentMap instance
func NewConcurrentMap[K comparable, T any]() *ConcurrentMap[K, T] {
	return &ConcurrentMap[K, T]{
		Map: make(map[K]T),
	}
}

// Get retrieves a value from the map
func (c *ConcurrentMap[K, T]) Get(key *K) (*T, bool) {
	c.Mutex.RLock()
	defer c.Mutex.RUnlock()
	val, ok := c.Map[*key]
	if !ok {
		return nil, false
	}
	return &val, true
}

// Set adds or updates a value in the map
func (c *ConcurrentMap[K, T]) Set(key *K, value *T) {
	c.Mutex.Lock()
	defer c.Mutex.Unlock()
	c.Map[*key] = *value
}

// Delete removes a key-value pair from the map
func (c *ConcurrentMap[K, T]) Delete(key *K) {
	c.Mutex.Lock()
	defer c.Mutex.Unlock()
	delete(c.Map, *key)
}

// Update modifies an existing value using a function for atomicity (hopefully)
// Return value is based on success
func (c *ConcurrentMap[K, T]) Update(key *K, fn func(*T) *T) bool {
	c.Mutex.Lock()
	defer c.Mutex.Unlock()
	if val, ok := c.Map[*key]; ok {
		c.Map[*key] = *fn(&val)
		return true
	}
	return false
}
