package search

import (
	"sync"
	"testing"
	"time"

	"yolo/caching"
)

// Test cache hit scenario
func TestSearchCacheHit(t *testing.T) {
	cache := caching.NewCaches(30 * time.Minute)
	query := "test query for cache"
	expectedResult := "cached content for test"

	// Pre-populate cache
	cache.Set(query, &caching.CachedResult{Content: expectedResult, CreatedAt: time.Now()})

	result, ok := cache.Get(query)
	if !ok {
		t.Error("Expected cache hit but got miss")
	}

	if result.Content != expectedResult {
		t.Errorf("Expected content '%s', got '%s'", expectedResult, result.Content)
	}
}

// Test cache miss scenario
func TestSearchCacheMiss(t *testing.T) {
	cache := caching.NewCaches(30 * time.Minute)
	query := "non-existent query"

	result, ok := cache.Get(query)
	if ok {
		t.Error("Expected cache miss but got hit")
	}
	if result != nil {
		t.Error("Expected nil result for cache miss")
	}
}

// Test CheckAndFetch with cache miss (should execute function)
func TestCheckAndFetchMiss(t *testing.T) {
	cache := caching.NewCaches(30 * time.Minute)
	query := "new query"
	fetchCalled := false
	expectedContent := "fetched content"

	result := cache.CheckAndFetch(query, func() interface{} {
		fetchCalled = true
		return expectedContent
	})

	if !fetchCalled {
		t.Error("Expected fetch function to be called on cache miss")
	}

	if result != expectedContent {
		t.Errorf("Expected result '%s', got '%s'", expectedContent, result)
	}

	// Verify it was cached
	_, ok := cache.Get(query)
	if !ok {
		t.Error("Expected result to be cached after CheckAndFetch")
	}
}

// Test CheckAndFetch with cache hit (should not execute function)
func TestCheckAndFetchHit(t *testing.T) {
	cache := caching.NewCaches(30 * time.Minute)
	query := "existing query"
	cachedContent := "already cached"
	fetchCalled := false

	cache.Set(query, &caching.CachedResult{Content: cachedContent, CreatedAt: time.Now()})

	result := cache.CheckAndFetch(query, func() interface{} {
		fetchCalled = true
		return "should not return this"
	})

	if fetchCalled {
		t.Error("Expected fetch function NOT to be called on cache hit")
	}

	if result != cachedContent {
		t.Errorf("Expected cached content '%s', got '%s'", cachedContent, result)
	}
}

// Test cache eviction of expired entries
func TestCacheEviction(t *testing.T) {
	cache := caching.NewCaches(5 * time.Minute)
	query := "expired query"
	expiredTime := time.Now().Add(-1 * time.Hour) // 1 hour ago

	cache.Set(query, &caching.CachedResult{Content: "old data", CreatedAt: expiredTime})

	result, ok := cache.Get(query)
	if ok {
		t.Error("Expected expired entry to be evicted")
	}
	if result != nil {
		t.Error("Expected nil result for evicted entry")
	}
}

// Test concurrent cache access
func TestConcurrentCacheAccess(t *testing.T) {
	cache := caching.NewCaches(30 * time.Minute)
	var wg sync.WaitGroup
	numGoroutines := 100

	// Concurrent writes
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			key := "concurrent-key-" + string(rune(id))
			cache.Set(key, &caching.CachedResult{Content: "value", CreatedAt: time.Now()})
		}(i)
	}

	// Concurrent reads
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			key := "concurrent-key-" + string(rune(id))
			cache.Get(key) // Should not panic
		}(i)
	}

	// Concurrent CheckAndFetch
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			key := "fetch-key-" + string(rune(id))
			cache.CheckAndFetch(key, func() interface{} {
				return "fetched"
			})
		}(i)
	}

	wg.Wait()
	// If we get here without panic, test passes
}

// Test GetStats returns correct information
func TestGetStats(t *testing.T) {
	cache := caching.NewCaches(30 * time.Minute)
	initialTime := time.Now()

	// Add some entries
	for i := 0; i < 10; i++ {
		key := "stats-key-" + string(rune(i))
		cache.Set(key, &caching.CachedResult{Content: "data", CreatedAt: initialTime})
	}

	stats := cache.GetStats()
	if stats.Size == -1 {
		t.Fatal("Expected non-nil stats")
	}

	if stats.Size != 10 {
		t.Errorf("Expected cache size 10, got %d", stats.Size)
	}
}

// Test setting and updating cached entries
func TestCacheUpdate(t *testing.T) {
	cache := caching.NewCaches(30 * time.Minute)
	query := "update test"
	timeBefore := time.Now()

	// First set
	cache.Set(query, &caching.CachedResult{Content: "first", CreatedAt: timeBefore})

	result1, _ := cache.Get(query)
	if result1.Content != "first" {
		t.Error("Expected 'first' content")
	}

	// Update with newer entry
	timeAfter := time.Now().Add(1 * time.Second)
	cache.Set(query, &caching.CachedResult{Content: "second", CreatedAt: timeAfter})

	result2, _ := cache.Get(query)
	if result2.Content != "second" {
		t.Error("Expected 'second' content after update")
	}
}

// Test empty cache operations
func TestEmptyCache(t *testing.T) {
	cache := caching.NewCaches(30 * time.Minute)

	result, ok := cache.Get("nonexistent")
	if ok || result != nil {
		t.Error("Empty cache should return miss")
	}

	stats := cache.GetStats()
	if stats.Size != 0 {
		t.Errorf("Empty cache size should be 0, got %d", stats.Size)
	}
}
