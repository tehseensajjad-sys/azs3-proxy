package cache

import (
	"os"
	"testing"
	"time"
)

// TestCacheStats_RecordOperations tests that cache stats correctly tracks operations
func TestCacheStats_RecordOperations(t *testing.T) {
	stats := &CacheStats{}

	// Test recording hits
	stats.RecordHit()
	stats.RecordHit()
	stats.RecordHit()

	// Test recording misses
	stats.RecordMiss()
	stats.RecordMiss()

	// Test recording puts
	stats.RecordPut(1024)
	stats.RecordPut(2048)

	// Test recording gets
	stats.RecordGet()
	stats.RecordGet()
	stats.RecordGet()

	// Test recording deletes
	stats.RecordDelete()

	// Test recording evictions
	stats.RecordEviction(512)
	stats.RecordEviction(1024)

	// Test recording expirations
	stats.RecordExpiration(256)
	stats.RecordExpiration(512)

	// Verify counts
	s := stats.GetStats()
	if s.Hits != 3 {
		t.Errorf("expected 3 hits, got %d", s.Hits)
	}
	if s.Misses != 2 {
		t.Errorf("expected 2 misses, got %d", s.Misses)
	}
	if s.Puts != 2 {
		t.Errorf("expected 2 puts, got %d", s.Puts)
	}
	if s.Gets != 3 {
		t.Errorf("expected 3 gets, got %d", s.Gets)
	}
	if s.Deletes != 1 {
		t.Errorf("expected 1 delete, got %d", s.Deletes)
	}
	if s.Evictions != 2 {
		t.Errorf("expected 2 evictions, got %d", s.Evictions)
	}
	if s.Expirations != 2 {
		t.Errorf("expected 2 expirations, got %d", s.Expirations)
	}
	if s.TotalBytesStored != 3072 { // 1024 + 2048
		t.Errorf("expected 3072 bytes stored, got %d", s.TotalBytesStored)
	}
	if s.TotalBytesEvicted != 2304 { // 512 + 1024 + 256 + 512 (evictions + expirations)
		t.Errorf("expected 2304 bytes evicted, got %d", s.TotalBytesEvicted)
	}
}

// TestCacheStats_HitRate tests that cache hit rate is calculated correctly
func TestCacheStats_HitRate(t *testing.T) {
	stats := &CacheStats{}

	// Test with 0 operations (edge case)
	s := stats.GetStats()
	if s.HitRate != 0.0 {
		t.Errorf("expected 0 hit rate with 0 operations, got %f", s.HitRate)
	}

	// Test with hits and misses
	stats.RecordHit()
	stats.RecordHit()
	stats.RecordMiss()
	stats.RecordMiss()
	stats.RecordMiss()

	s = stats.GetStats()
	expectedHitRate := (2.0 / 5.0) * 100 // 2 hits out of 5 total, expressed as percentage
	if s.HitRate != expectedHitRate {
		t.Errorf("expected hit rate of %f, got %f", expectedHitRate, s.HitRate)
	}

	// Test 100% hit rate
	stats2 := &CacheStats{}
	for i := 0; i < 10; i++ {
		stats2.RecordHit()
	}
	s2 := stats2.GetStats()
	if s2.HitRate != 100.0 {
		t.Errorf("expected 100 hit rate, got %f", s2.HitRate)
	}
}

// TestCacheStats_Reset tests that stats can be reset
func TestCacheStats_Reset(t *testing.T) {
	stats := &CacheStats{}

	// Record some operations
	stats.RecordHit()
	stats.RecordHit()
	stats.RecordMiss()
	stats.RecordPut(1024)
	stats.RecordEviction(512)

	// Verify stats are recorded
	s := stats.GetStats()
	if s.Hits != 2 || s.Misses != 1 || s.Puts != 1 || s.Evictions != 1 {
		t.Error("stats not properly recorded before reset")
	}

	// Reset stats
	stats.Reset()

	// Verify all stats are cleared
	s = stats.GetStats()
	if s.Hits != 0 || s.Misses != 0 || s.Puts != 0 || s.Gets != 0 ||
		s.Deletes != 0 || s.Evictions != 0 || s.Expirations != 0 ||
		s.TotalBytesStored != 0 || s.TotalBytesEvicted != 0 {
		t.Error("stats not properly reset")
	}
}

// TestLRUCache_GetStats tests that LRUCache exposes stats correctly
func TestLRUCache_GetStats(t *testing.T) {
	tempDir := t.TempDir()
	cache, err := NewLRUCache(tempDir, 10*1024, 30*time.Second)
	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}
	defer cache.Close()

	// Create a test file to cache
	testFile := tempDir + "/test-source.txt"
	testData := []byte("test data for caching")
	if err := os.WriteFile(testFile, testData, 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Cache the file
	if err := cache.Put("test-key", testFile, int64(len(testData))); err != nil {
		t.Fatalf("failed to cache file: %v", err)
	}

	// Retrieve from cache (hit)
	_, err = cache.Get("test-key")
	if err != nil {
		t.Fatalf("failed to get from cache: %v", err)
	}

	// Try to retrieve non-existent key (miss)
	_, err = cache.Get("non-existent-key")
	if err != nil {
		// Expected: key not found
	}

	// Get stats
	stats := cache.GetStats()

	if stats.Hits != 1 {
		t.Errorf("expected 1 hit, got %d", stats.Hits)
	}
	if stats.Misses < 1 {
		t.Errorf("expected at least 1 miss, got %d", stats.Misses)
	}
	if stats.Puts != 1 {
		t.Errorf("expected 1 put, got %d", stats.Puts)
	}
	if stats.TotalBytesStored != int64(len(testData)) {
		t.Errorf("expected %d bytes stored, got %d", len(testData), stats.TotalBytesStored)
	}
}

// TestCacheManager_GetStats tests that CacheManager exposes stats
func TestCacheManager_GetStats(t *testing.T) {
	tempDir := t.TempDir()
	manager, err := NewCacheManager(tempDir, 10*1024, 30)
	if err != nil {
		t.Fatalf("failed to create cache manager: %v", err)
	}
	defer manager.Close()

	// Cache an object
	testData := []byte("test object data")
	if err := manager.CacheObject("object-key", testData); err != nil {
		t.Fatalf("failed to cache object: %v", err)
	}

	// Read from cache
	_, err = manager.ReadCachedObject("object-key")
	if err != nil {
		t.Fatalf("failed to read cached object: %v", err)
	}

	// Try to read non-existent object
	_, _ = manager.ReadCachedObject("non-existent")

	// Get stats
	stats := manager.GetStats()

	if stats.Puts < 1 {
		t.Errorf("expected at least 1 put, got %d", stats.Puts)
	}
	if stats.Hits < 1 {
		t.Errorf("expected at least 1 hit, got %d", stats.Hits)
	}
}

// TestCacheStats_ConcurrentUpdates tests that stats are thread-safe
func TestCacheStats_ConcurrentUpdates(t *testing.T) {
	stats := &CacheStats{}

	// Run multiple goroutines updating stats concurrently
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				stats.RecordHit()
				stats.RecordPut(int64(j))
				stats.RecordEviction(int64(j / 2))
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify final counts
	s := stats.GetStats()
	if s.Hits != 1000 { // 10 goroutines * 100 hits
		t.Errorf("expected 1000 hits, got %d", s.Hits)
	}
	if s.Puts != 1000 {
		t.Errorf("expected 1000 puts, got %d", s.Puts)
	}
	if s.Evictions != 1000 {
		t.Errorf("expected 1000 evictions, got %d", s.Evictions)
	}
}
