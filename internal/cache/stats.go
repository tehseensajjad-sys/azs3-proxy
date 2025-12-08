package cache

import (
	"sync"
	"sync/atomic"
)

// CacheStats tracks performance metrics and operational statistics for the cache.
// All operations are thread-safe for concurrent access.
type CacheStats struct {
	// Hit/Miss tracking
	hits   atomic.Int64 // Number of successful cache hits
	misses atomic.Int64 // Number of cache misses

	// Eviction tracking
	evictions   atomic.Int64 // Number of entries evicted due to LRU policy
	expirations atomic.Int64 // Number of entries removed due to TTL expiration

	// Size tracking
	totalBytesStored  atomic.Int64 // Total bytes ever stored in cache
	totalBytesEvicted atomic.Int64 // Total bytes removed from cache

	// Operation tracking
	puts    atomic.Int64 // Total Put operations
	gets    atomic.Int64 // Total Get operations
	deletes atomic.Int64 // Total Delete operations

	mutex sync.RWMutex
}

// RecordHit increments the cache hit counter.
func (cs *CacheStats) RecordHit() {
	cs.hits.Add(1)
}

// RecordMiss increments the cache miss counter.
func (cs *CacheStats) RecordMiss() {
	cs.misses.Add(1)
}

// RecordEviction increments the eviction counter and updates bytes evicted.
func (cs *CacheStats) RecordEviction(bytes int64) {
	cs.evictions.Add(1)
	cs.totalBytesEvicted.Add(bytes)
}

// RecordExpiration increments the expiration counter and updates bytes evicted.
func (cs *CacheStats) RecordExpiration(bytes int64) {
	cs.expirations.Add(1)
	cs.totalBytesEvicted.Add(bytes)
}

// RecordPut increments the Put operation counter and updates total bytes stored.
func (cs *CacheStats) RecordPut(bytes int64) {
	cs.puts.Add(1)
	cs.totalBytesStored.Add(bytes)
}

// RecordGet increments the Get operation counter.
func (cs *CacheStats) RecordGet() {
	cs.gets.Add(1)
}

// RecordDelete increments the Delete operation counter.
func (cs *CacheStats) RecordDelete() {
	cs.deletes.Add(1)
}

// GetStats returns a snapshot of current cache statistics.
// It calculates derived metrics like hit rate and average entry size.
func (cs *CacheStats) GetStats() Statistics {
	hits := cs.hits.Load()
	misses := cs.misses.Load()
	totalRequests := hits + misses

	hitRate := float64(0)
	if totalRequests > 0 {
		hitRate = float64(hits) / float64(totalRequests) * 100
	}

	return Statistics{
		Hits:              hits,
		Misses:            misses,
		TotalRequests:     totalRequests,
		HitRate:           hitRate,
		Evictions:         cs.evictions.Load(),
		Expirations:       cs.expirations.Load(),
		TotalBytesStored:  cs.totalBytesStored.Load(),
		TotalBytesEvicted: cs.totalBytesEvicted.Load(),
		Puts:              cs.puts.Load(),
		Gets:              cs.gets.Load(),
		Deletes:           cs.deletes.Load(),
	}
}

// Reset clears all statistics counters.
func (cs *CacheStats) Reset() {
	cs.mutex.Lock()
	defer cs.mutex.Unlock()

	cs.hits.Store(0)
	cs.misses.Store(0)
	cs.evictions.Store(0)
	cs.expirations.Store(0)
	cs.totalBytesStored.Store(0)
	cs.totalBytesEvicted.Store(0)
	cs.puts.Store(0)
	cs.gets.Store(0)
	cs.deletes.Store(0)
}

// Statistics is a snapshot of cache performance metrics.
// Includes hit/miss rates, eviction counts, and operation counts.
type Statistics struct {
	// Hit/Miss metrics
	Hits          int64   // Total cache hits
	Misses        int64   // Total cache misses
	TotalRequests int64   // Total Get requests (hits + misses)
	HitRate       float64 // Hit rate as percentage (0-100)

	// Eviction metrics
	Evictions   int64 // Total LRU evictions
	Expirations int64 // Total TTL expirations

	// Byte metrics
	TotalBytesStored  int64 // Total bytes ever stored
	TotalBytesEvicted int64 // Total bytes ever evicted

	// Operation counts
	Puts    int64 // Total Put operations
	Gets    int64 // Total Get operations
	Deletes int64 // Total Delete operations
}
