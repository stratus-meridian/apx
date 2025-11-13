package policy

import (
	"fmt"
	"sync"
	"time"
)

// CacheEntry represents a cached policy version
type CacheEntry struct {
	Name       string
	Version    string
	Hash       string
	WASM       []byte
	LoadedAt   time.Time
	LastUsedAt time.Time
}

// Cache manages policy versions with N/N-1 support
type Cache struct {
	mu      sync.RWMutex
	entries map[string]*CacheEntry // key: {name}@{version}
	ttl     time.Duration          // TTL for N-1 versions
}

// NewCache creates a new policy cache
func NewCache(ttl time.Duration) *Cache {
	return &Cache{
		entries: make(map[string]*CacheEntry),
		ttl:     ttl,
	}
}

// Get retrieves a policy from cache
func (c *Cache) Get(name, version string) (*CacheEntry, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	key := fmt.Sprintf("%s@%s", name, version)
	entry, exists := c.entries[key]
	if !exists {
		return nil, false
	}

	// Update last used time
	entry.LastUsedAt = time.Now()
	return entry, true
}

// Set adds or updates a policy in cache
func (c *Cache) Set(entry *CacheEntry) {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := fmt.Sprintf("%s@%s", entry.Name, entry.Version)
	entry.LoadedAt = time.Now()
	entry.LastUsedAt = time.Now()
	c.entries[key] = entry
}

// Evict removes expired entries (N-1 versions older than TTL)
func (c *Cache) Evict() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	evicted := 0

	for key, entry := range c.entries {
		if now.Sub(entry.LoadedAt) > c.ttl {
			delete(c.entries, key)
			evicted++
		}
	}

	return evicted
}

// Size returns the number of cached entries
func (c *Cache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.entries)
}

// Keys returns all cache keys
func (c *Cache) Keys() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	keys := make([]string, 0, len(c.entries))
	for k := range c.entries {
		keys = append(keys, k)
	}
	return keys
}
