package caching

import (
	"sync"
	"time"
)

// CachedResult represents a cached value with metadata
type CachedResult struct {
	Content   interface{}
	CreatedAt time.Time
}

// Caches provides thread-safe caching with TTL support
type Caches struct {
	mu      sync.RWMutex
	items   map[string]*CachedResult
	ttl     time.Duration
	started time.Time
}

// NewCaches creates a new cache with the specified TTL
func NewCaches(ttl time.Duration) *Caches {
	return &Caches{
		items:   make(map[string]*CachedResult),
		ttl:     ttl,
		started: time.Now(),
	}
}

// isExpired checks if a cached entry has expired
func (c *Caches) isExpired(entry *CachedResult) bool {
	return time.Since(entry.CreatedAt) > c.ttl
}

// Get retrieves a value from the cache, returning nil and false if not found or expired
func (c *Caches) Get(key string) (*CachedResult, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.items[key]
	if !ok {
		return nil, false
	}

	if c.isExpired(entry) {
		delete(c.items, key)
		return nil, false
	}

	return entry, true
}

// Set stores a value in the cache
func (c *Caches) Set(key string, entry *CachedResult) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Clean up expired entries before setting new one
	c.cleanup()

	c.items[key] = entry
}

// CheckAndFetch retrieves a value from cache or computes it using the fetch function
func (c *Caches) CheckAndFetch(key string, fetchFunc func() interface{}) interface{} {
	// Try to get from cache first
	if result, ok := c.Get(key); ok {
		return result.Content
	}

	// Cache miss - compute and store
	c.mu.Lock()
	defer c.mu.Unlock()

	// Double-check after acquiring write lock
	if entry, ok := c.items[key]; ok && !c.isExpired(entry) {
		return entry.Content
	}

	// Compute value
	result := fetchFunc()

	// Store in cache
	c.items[key] = &CachedResult{
		Content:   result,
		CreatedAt: time.Now(),
	}

	return result
}

// cleanup removes expired entries from the cache
func (c *Caches) cleanup() {
	now := time.Now()
	for key, entry := range c.items {
		if now.Sub(entry.CreatedAt) > c.ttl {
			delete(c.items, key)
		}
	}
}

// GetStats returns statistics about the cache
func (c *Caches) GetStats() CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	stats := CacheStats{
		Size:      len(c.items),
		StartedAt: c.started,
		Uptime:    time.Since(c.started),
	}

	// Count expired entries
	expiredCount := 0
	for _, entry := range c.items {
		if time.Since(entry.CreatedAt) > c.ttl {
			expiredCount++
		}
	}
	stats.Expired = expiredCount

	return stats
}

// CacheStats holds cache statistics
type CacheStats struct {
	Size      int
	StartedAt time.Time
	Uptime    time.Duration
	Expired   int
}
