package cache

import (
	"container/list"
	"sync"
	"time"
)

type entry struct {
	key       string
	value     any
	expiresAt time.Time
}

// LRUCache is a thread-safe LRU cache with TTL support
type LRUCache struct {
	capacity int
	ttl      time.Duration
	mu       sync.RWMutex
	items    map[string]*list.Element
	lru      *list.List
}

// NewLRUCache creates a new LRU cache with specified capacity and TTL
func NewLRUCache(capacity int, ttl time.Duration) *LRUCache {
	return &LRUCache{
		capacity: capacity,
		ttl:      ttl,
		items:    make(map[string]*list.Element),
		lru:      list.New(),
	}
}

// Get retrieves a value from the cache
func (c *LRUCache) Get(key string) (any, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	elem, exists := c.items[key]
	if !exists {
		return nil, false
	}

	entry := elem.Value.(*entry)

	// Check if expired
	if time.Now().After(entry.expiresAt) {
		c.removeElement(elem)
		return nil, false
	}

	// Move to front (most recently used)
	c.lru.MoveToFront(elem)
	return entry.value, true
}

// Set adds or updates a value in the cache
func (c *LRUCache) Set(key string, value any) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// If key exists, update it
	if elem, exists := c.items[key]; exists {
		c.lru.MoveToFront(elem)
		entry := elem.Value.(*entry)
		entry.value = value
		entry.expiresAt = time.Now().Add(c.ttl)
		return
	}

	// Add new entry
	newEntry := &entry{
		key:       key,
		value:     value,
		expiresAt: time.Now().Add(c.ttl),
	}
	elem := c.lru.PushFront(newEntry)
	c.items[key] = elem

	// Evict if over capacity
	if c.lru.Len() > c.capacity {
		c.removeOldest()
	}
}

// removeOldest removes the least recently used item
func (c *LRUCache) removeOldest() {
	elem := c.lru.Back()
	if elem != nil {
		c.removeElement(elem)
	}
}

// removeElement removes a specific element
func (c *LRUCache) removeElement(elem *list.Element) {
	c.lru.Remove(elem)
	entry := elem.Value.(*entry)
	delete(c.items, entry.key)
}

// Len returns the number of items in the cache
func (c *LRUCache) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.lru.Len()
}

// Clear removes all items from the cache
func (c *LRUCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items = make(map[string]*list.Element)
	c.lru = list.New()
}
