package cache

import (
	"sync"
	"time"
)

// Item represents a cached item
type Item struct {
	Value      []byte
	Expiration int64
}

// Cache is an in-memory cache
type Cache struct {
	items map[string]Item
	mu    sync.RWMutex
}

// New creates a new cache
func New() *Cache {
	cache := &Cache{
		items: make(map[string]Item),
	}

	// Start the janitor to clean up expired items
	go cache.janitor()

	return cache
}

// Set adds an item to the cache with the given key and expiration
func (c *Cache) Set(key string, value []byte, expiration time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	var exp int64
	if expiration > 0 {
		exp = time.Now().Add(expiration).UnixNano()
	}

	c.items[key] = Item{
		Value:      value,
		Expiration: exp,
	}
}

// Get retrieves an item from the cache
func (c *Cache) Get(key string) ([]byte, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, found := c.items[key]
	if !found {
		return nil, false
	}

	// Check if the item has expired
	if item.Expiration > 0 && time.Now().UnixNano() > item.Expiration {
		return nil, false
	}

	return item.Value, true
}

// Delete removes an item from the cache
func (c *Cache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.items, key)
}

// Clear removes all items from the cache
func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items = make(map[string]Item)
}

// janitor cleans up expired items from the cache
func (c *Cache) janitor() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		c.deleteExpired()
	}
}

// deleteExpired deletes expired items from the cache
func (c *Cache) deleteExpired() {
	now := time.Now().UnixNano()

	c.mu.Lock()
	defer c.mu.Unlock()

	for k, v := range c.items {
		if v.Expiration > 0 && now > v.Expiration {
			delete(c.items, k)
		}
	}
}