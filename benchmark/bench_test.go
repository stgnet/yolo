package benchmark

import (
	"testing"
	"time"

	"yolo/caching"
)

// BenchmarkSearchCache stores performance metrics
func BenchmarkSearchCacheMiss(b *testing.B) {
	cache := caching.NewCaches(30 * time.Minute)
	query := "test query for benchmarking purposes"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Simulate cache miss (new query each time)
		key := query + "-miss-" + string(rune(i))
		cache.CheckAndFetch(key, func() interface{} {
			return "result"
		})
	}
}

func BenchmarkSearchCacheHit(b *testing.B) {
	cache := caching.NewCaches(30 * time.Minute)
	query := "test query for benchmarking purposes"

	// Pre-populate cache
	result := "cached result data with some content to make it more realistic"
	cache.Set(query, &caching.CachedResult{Content: result, CreatedAt: time.Now()})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.CheckAndFetch(query, func() interface{} {
			return "should not be called"
		})
	}
}

func BenchmarkSearchCacheEviction(b *testing.B) {
	cache := caching.NewCaches(5 * time.Minute) // Short TTL for testing

	// Add expired entries
	for i := 0; i < 100; i++ {
		key := "expired-key-" + string(rune(i))
		cache.Set(key, &caching.CachedResult{Content: "old data", CreatedAt: time.Now().Add(-1 * time.Hour)})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// This should trigger cleanup
		cache.GetStats()
	}
}

// Benchmark for the entire yolo package
func TestBenchmarks(t *testing.T) {
	t.Log("Running all benchmarks - use 'go test -bench=. ./...' to execute")
}
