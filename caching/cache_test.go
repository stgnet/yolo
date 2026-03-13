package caching

import (
	"sync"
	"testing"
	"time"
)

func TestNewCaches(t *testing.T) {
	cache := NewCaches(5 * time.Minute)
	if cache == nil {
		t.Fatal("Expected non-nil cache")
	}
	if cache.ttl != 5*time.Minute {
		t.Errorf("Expected TTL 5 minutes, got %v", cache.ttl)
	}
	if cache.items == nil {
		t.Error("Expected items map to be initialized")
	}
}

func TestGetAndSet(t *testing.T) {
	cache := NewCaches(30 * time.Minute)
	key := "test-key"
	expectedValue := &CachedResult{Content: "test data", CreatedAt: time.Now()}

	cache.Set(key, expectedValue)

	result, ok := cache.Get(key)
	if !ok {
		t.Fatal("Expected cache hit")
	}
	if result != expectedValue {
		t.Error("Expected same CachedResult pointer")
	}
	if result.Content != "test data" {
		t.Errorf("Expected content 'test data', got '%v'", result.Content)
	}
}

func TestGetNonExistent(t *testing.T) {
	cache := NewCaches(30 * time.Minute)

	result, ok := cache.Get("non-existent-key")
	if ok {
		t.Error("Expected cache miss for non-existent key")
	}
	if result != nil {
		t.Error("Expected nil result for non-existent key")
	}
}

func TestCheckAndFetchMiss(t *testing.T) {
	cache := NewCaches(30 * time.Minute)
	fetchCalled := false

	result := cache.CheckAndFetch("new-key", func() interface{} {
		fetchCalled = true
		return "fetched value"
	})

	if !fetchCalled {
		t.Error("Expected fetch function to be called")
	}
	if result != "fetched value" {
		t.Errorf("Expected 'fetched value', got '%v'", result)
	}

	// Verify it was cached
	cached, ok := cache.Get("new-key")
	if !ok {
		t.Error("Expected key to be cached after CheckAndFetch")
	}
	if cached.Content != "fetched value" {
		t.Errorf("Expected cached content 'fetched value', got '%v'", cached.Content)
	}
}

func TestCheckAndFetchHit(t *testing.T) {
	cache := NewCaches(30 * time.Minute)
	cache.Set("existing-key", &CachedResult{Content: "cached-value", CreatedAt: time.Now()})
	fetchCalled := false

	result := cache.CheckAndFetch("existing-key", func() interface{} {
		fetchCalled = true
		return "should-not-return-this"
	})

	if fetchCalled {
		t.Error("Expected fetch function NOT to be called on cache hit")
	}
	if result != "cached-value" {
		t.Errorf("Expected 'cached-value', got '%v'", result)
	}
}

func TestCacheExpiration(t *testing.T) {
	cache := NewCaches(100 * time.Millisecond) // Short TTL for testing
	key := "expiring-key"

	cache.Set(key, &CachedResult{Content: "temp-data", CreatedAt: time.Now()})

	// Should exist initially
	if _, ok := cache.Get(key); !ok {
		t.Error("Expected key to exist before expiration")
	}

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Should be expired now
	result, ok := cache.Get(key)
	if ok {
		t.Error("Expected expired key to return false")
	}
	if result != nil {
		t.Error("Expected nil result for expired key")
	}
}

func TestCacheEvictionOnSet(t *testing.T) {
	cache := NewCaches(100 * time.Millisecond)

	// Add expired entry
	expiredTime := time.Now().Add(-1 * time.Second)
	cache.Set("expired-key", &CachedResult{Content: "old", CreatedAt: expiredTime})

	// Verify it's gone (evicted during Set)
	if _, ok := cache.Get("expired-key"); ok {
		t.Error("Expected expired entry to be evicted")
	}

	// Add new entry and verify it works
	cache.Set("new-key", &CachedResult{Content: "new", CreatedAt: time.Now()})
	result, ok := cache.Get("new-key")
	if !ok {
		t.Error("Expected new key to exist")
	}
	if result.Content != "new" {
		t.Errorf("Expected 'new', got '%v'", result.Content)
	}
}

func TestConcurrentAccess(t *testing.T) {
	cache := NewCaches(30 * time.Minute)
	var wg sync.WaitGroup
	numGoroutines := 100

	// Concurrent writes
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			key := "key-" + string(rune(id))
			cache.Set(key, &CachedResult{Content: "value", CreatedAt: time.Now()})
		}(i)
	}

	// Concurrent reads
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			key := "key-" + string(rune(id))
			cache.Get(key) // Should not panic or race
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
	// Test passes if no panic or race detected
}

func TestGetStats(t *testing.T) {
	cache := NewCaches(30 * time.Minute)
	startTime := cache.started

	// Add entries
	for i := 0; i < 5; i++ {
		cache.Set("key-"+string(rune(i)), &CachedResult{Content: "data", CreatedAt: time.Now()})
	}

	stats := cache.GetStats()
	if stats.Size != 5 {
		t.Errorf("Expected size 5, got %d", stats.Size)
	}
	if stats.StartedAt.Before(startTime) || stats.StartedAt.After(startTime.Add(1*time.Second)) {
		t.Error("Expected StartedAt to be approximately start time")
	}
	if stats.Uptime <= 0 {
		t.Error("Expected positive uptime")
	}
	if stats.Expired != 0 {
		t.Errorf("Expected 0 expired entries, got %d", stats.Expired)
	}
}

func TestGetStatsWithExpired(t *testing.T) {
	cache := NewCaches(100 * time.Millisecond)

	// Add expired entry
	expiredTime := time.Now().Add(-500 * time.Millisecond)
	cache.items["expired"] = &CachedResult{Content: "old", CreatedAt: expiredTime}

	// Add fresh entry
	cache.items["fresh"] = &CachedResult{Content: "new", CreatedAt: time.Now()}

	stats := cache.GetStats()
	if stats.Size != 2 {
		t.Errorf("Expected size 2, got %d", stats.Size)
	}
	if stats.Expired != 1 {
		t.Errorf("Expected 1 expired entry, got %d", stats.Expired)
	}
}

func TestUpdateExistingKey(t *testing.T) {
	cache := NewCaches(30 * time.Minute)
	key := "update-key"

	// Initial set
	cache.Set(key, &CachedResult{Content: "first", CreatedAt: time.Now()})
	result1, _ := cache.Get(key)
	if result1.Content != "first" {
		t.Error("Expected 'first' content")
	}

	// Update
	cache.Set(key, &CachedResult{Content: "second", CreatedAt: time.Now()})
	result2, _ := cache.Get(key)
	if result2.Content != "second" {
		t.Error("Expected 'second' content after update")
	}
}

func TestEmptyCacheOperations(t *testing.T) {
	cache := NewCaches(30 * time.Minute)

	// Get on empty cache
	result, ok := cache.Get("nonexistent")
	if ok {
		t.Error("Expected false for non-existent key in empty cache")
	}
	if result != nil {
		t.Error("Expected nil result for non-existent key")
	}

	// CheckAndFetch on empty cache should work
	resultValue := cache.CheckAndFetch("new", func() interface{} {
		return "computed"
	})
	if resultValue != "computed" {
		t.Errorf("Expected 'computed', got '%v'", resultValue)
	}

	// Stats on empty (initially) cache should work
	stats := cache.GetStats()
	if stats.Size != 1 { // After CheckAndFetch added one entry
		t.Errorf("Expected size 1 after CheckAndFetch, got %d", stats.Size)
	}
}

func TestCacheCleanup(t *testing.T) {
	cache := NewCaches(30 * time.Second) // Long TTL so fresh entries don't expire during test

	// Add mix of expired and fresh entries directly (bypassing Set to control timing)
	expiredTime := time.Now().Add(-1 * time.Hour)
	freshTime := time.Now()

	cache.items["key-0"] = &CachedResult{Content: "old", CreatedAt: expiredTime}
	cache.items["key-1"] = &CachedResult{Content: "new", CreatedAt: freshTime}
	cache.items["key-2"] = &CachedResult{Content: "old", CreatedAt: expiredTime}

	// Trigger cleanup via Set (this will clean up expired entries)
	cache.Set("trigger", &CachedResult{Content: "trigger", CreatedAt: time.Now()})

	// Verify expired entries were removed
	if _, ok := cache.Get("key-0"); ok {
		t.Error("Expected expired entry key-0 to be removed")
	}
	if _, ok := cache.Get("key-2"); ok {
		t.Error("Expected expired entry key-2 to be removed")
	}

	// Verify fresh entries still exist
	if result, ok := cache.Get("key-1"); !ok {
		t.Error("Expected fresh entry key-1 to remain")
	} else if result.Content != "new" {
		t.Errorf("Expected key-1 content 'new', got '%v'", result.Content)
	}

	if _, ok := cache.Get("trigger"); !ok {
		t.Error("Expected trigger entry to exist")
	}
}
