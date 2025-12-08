package cache

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// TestNewLRUCache tests that NewLRUCache creates a cache with correct initialization.
func TestNewLRUCache(t *testing.T) {
	cacheDir := t.TempDir()
	maxBytes := int64(1024 * 1024) // 1MB
	ttl := 5 * time.Minute

	cache, err := NewLRUCache(cacheDir, maxBytes, ttl)
	if err != nil {
		t.Fatalf("NewLRUCache failed: %v", err)
	}
	defer cache.Close()

	if cache.Size() != 0 {
		t.Errorf("Expected cache size 0, got %d", cache.Size())
	}

	if cache.Count() != 0 {
		t.Errorf("Expected cache count 0, got %d", cache.Count())
	}

	// Verify directory was created
	if _, err := os.Stat(cacheDir); err != nil {
		t.Errorf("Cache directory not created: %v", err)
	}
}

// TestPutAndGet tests basic Put and Get operations.
func TestPutAndGet(t *testing.T) {
	cacheDir := t.TempDir()
	maxBytes := int64(1024 * 1024)
	ttl := 5 * time.Minute

	cache, err := NewLRUCache(cacheDir, maxBytes, ttl)
	if err != nil {
		t.Fatalf("NewLRUCache failed: %v", err)
	}
	defer cache.Close()

	// Create a test file
	testFile := filepath.Join(t.TempDir(), "test.txt")
	testData := []byte("hello world")
	if err := os.WriteFile(testFile, testData, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Put file in cache
	key := "test-key"
	if err := cache.Put(key, testFile, int64(len(testData))); err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	// Verify cache size updated
	if cache.Size() != int64(len(testData)) {
		t.Errorf("Expected cache size %d, got %d", len(testData), cache.Size())
	}

	// Get file from cache
	cachedPath, err := cache.Get(key)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if cachedPath == "" {
		t.Fatal("Get returned empty path")
	}

	// Verify cached file content
	content, err := os.ReadFile(cachedPath)
	if err != nil {
		t.Fatalf("Failed to read cached file: %v", err)
	}

	if string(content) != string(testData) {
		t.Errorf("Cached file content mismatch. Expected %s, got %s", testData, content)
	}
}

// TestLRUEviction tests that oldest entries are evicted when cache exceeds maxBytes.
func TestLRUEviction(t *testing.T) {
	cacheDir := t.TempDir()
	fileSize := int64(100)
	maxBytes := fileSize * 3 // Space for 3 files
	ttl := 5 * time.Minute

	cache, err := NewLRUCache(cacheDir, maxBytes, ttl)
	if err != nil {
		t.Fatalf("NewLRUCache failed: %v", err)
	}
	defer cache.Close()

	// Create and cache 3 files
	keys := make([]string, 3)
	for i := 0; i < 3; i++ {
		testFile := filepath.Join(t.TempDir(), fmt.Sprintf("test%d.txt", i))
		testData := make([]byte, fileSize)
		if err := os.WriteFile(testFile, testData, 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		key := fmt.Sprintf("key-%d", i)
		keys[i] = key
		if err := cache.Put(key, testFile, fileSize); err != nil {
			t.Fatalf("Put failed: %v", err)
		}

		// Small delay to ensure different timestamps
		time.Sleep(10 * time.Millisecond)
	}

	if cache.Count() != 3 {
		t.Errorf("Expected 3 entries, got %d", cache.Count())
	}

	// Add 4th file - should evict the oldest (key-0)
	testFile := filepath.Join(t.TempDir(), "test4.txt")
	testData := make([]byte, fileSize)
	if err := os.WriteFile(testFile, testData, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	if err := cache.Put("key-4", testFile, fileSize); err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	if cache.Count() != 3 {
		t.Errorf("Expected 3 entries after eviction, got %d", cache.Count())
	}

	// Verify oldest key was evicted
	if cache.Exists(keys[0]) {
		t.Error("Expected key-0 to be evicted")
	}

	// Verify newest keys still exist
	if !cache.Exists(keys[1]) {
		t.Error("Expected key-1 to still exist")
	}
	if !cache.Exists(keys[2]) {
		t.Error("Expected key-2 to still exist")
	}
	if !cache.Exists("key-4") {
		t.Error("Expected key-4 to exist")
	}
}

// TestTTLExpiration tests that entries expire after TTL.
func TestTTLExpiration(t *testing.T) {
	cacheDir := t.TempDir()
	maxBytes := int64(1024 * 1024)
	ttl := 100 * time.Millisecond // Short TTL for testing

	cache, err := NewLRUCache(cacheDir, maxBytes, ttl)
	if err != nil {
		t.Fatalf("NewLRUCache failed: %v", err)
	}
	defer cache.Close()

	// Create and cache a file
	testFile := filepath.Join(t.TempDir(), "test.txt")
	testData := []byte("test data")
	if err := os.WriteFile(testFile, testData, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	key := "test-key"
	if err := cache.Put(key, testFile, int64(len(testData))); err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	// Verify entry exists
	if !cache.Exists(key) {
		t.Error("Entry should exist immediately after Put")
	}

	// Wait for entry to expire
	time.Sleep(ttl + 50*time.Millisecond)

	// Verify entry has expired
	if cache.Exists(key) {
		t.Error("Entry should have expired after TTL")
	}

	// Verify Get returns nil
	cachedPath, err := cache.Get(key)
	if cachedPath != "" {
		t.Errorf("Get should return empty path for expired entry, got %s", cachedPath)
	}
}

// TestDelete tests that Delete removes entries correctly.
func TestDelete(t *testing.T) {
	cacheDir := t.TempDir()
	maxBytes := int64(1024 * 1024)
	ttl := 5 * time.Minute

	cache, err := NewLRUCache(cacheDir, maxBytes, ttl)
	if err != nil {
		t.Fatalf("NewLRUCache failed: %v", err)
	}
	defer cache.Close()

	// Create and cache a file
	testFile := filepath.Join(t.TempDir(), "test.txt")
	testData := []byte("test data")
	if err := os.WriteFile(testFile, testData, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	key := "test-key"
	if err := cache.Put(key, testFile, int64(len(testData))); err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	// Verify entry exists
	if !cache.Exists(key) {
		t.Error("Entry should exist after Put")
	}

	initialSize := cache.Size()

	// Delete entry
	if err := cache.Delete(key); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify entry is gone
	if cache.Exists(key) {
		t.Error("Entry should not exist after Delete")
	}

	// Verify size decreased
	if cache.Size() != initialSize-int64(len(testData)) {
		t.Errorf("Cache size should decrease after Delete")
	}
}

// TestClear tests that Clear removes all entries.
func TestClear(t *testing.T) {
	cacheDir := t.TempDir()
	maxBytes := int64(1024 * 1024)
	ttl := 5 * time.Minute

	cache, err := NewLRUCache(cacheDir, maxBytes, ttl)
	if err != nil {
		t.Fatalf("NewLRUCache failed: %v", err)
	}
	defer cache.Close()

	// Create and cache multiple files
	for i := 0; i < 3; i++ {
		testFile := filepath.Join(t.TempDir(), fmt.Sprintf("test%d.txt", i))
		testData := []byte("test data")
		if err := os.WriteFile(testFile, testData, 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		key := fmt.Sprintf("key-%d", i)
		if err := cache.Put(key, testFile, int64(len(testData))); err != nil {
			t.Fatalf("Put failed: %v", err)
		}
	}

	if cache.Count() != 3 {
		t.Errorf("Expected 3 entries, got %d", cache.Count())
	}

	// Clear cache
	if err := cache.Clear(); err != nil {
		t.Fatalf("Clear failed: %v", err)
	}

	// Verify all entries are gone
	if cache.Count() != 0 {
		t.Errorf("Expected 0 entries after Clear, got %d", cache.Count())
	}

	if cache.Size() != 0 {
		t.Errorf("Expected cache size 0 after Clear, got %d", cache.Size())
	}
}

// TestConcurrentAccess tests thread-safe concurrent access to cache.
func TestConcurrentAccess(t *testing.T) {
	cacheDir := t.TempDir()
	maxBytes := int64(10 * 1024 * 1024) // 10MB
	ttl := 5 * time.Minute

	cache, err := NewLRUCache(cacheDir, maxBytes, ttl)
	if err != nil {
		t.Fatalf("NewLRUCache failed: %v", err)
	}
	defer cache.Close()

	// Create test files
	testFiles := make(map[int]string)
	for i := 0; i < 10; i++ {
		testFile := filepath.Join(t.TempDir(), fmt.Sprintf("test%d.txt", i))
		testData := make([]byte, 1024) // 1KB each
		if err := os.WriteFile(testFile, testData, 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
		testFiles[i] = testFile
	}

	// Concurrent puts and gets
	var wg sync.WaitGroup
	numGoroutines := 10

	for g := 0; g < numGoroutines; g++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			// Each goroutine puts and gets different files
			for i := 0; i < 10; i++ {
				key := fmt.Sprintf("key-%d-%d", id, i)
				fileIdx := (id + i) % 10

				if err := cache.Put(key, testFiles[fileIdx], 1024); err != nil {
					t.Errorf("Concurrent Put failed: %v", err)
				}

				if cachedPath, err := cache.Get(key); err != nil {
					t.Errorf("Concurrent Get failed: %v", err)
				} else if cachedPath == "" {
					t.Errorf("Concurrent Get returned empty path")
				}
			}
		}(g)
	}

	wg.Wait()

	// Cache should have some entries
	if cache.Count() == 0 {
		t.Error("Cache should have entries after concurrent operations")
	}

	if cache.Size() == 0 {
		t.Error("Cache size should be non-zero after concurrent operations")
	}
}

// TestLRUOrdering tests that LRU ordering is properly maintained.
func TestLRUOrdering(t *testing.T) {
	cacheDir := t.TempDir()
	fileSize := int64(100)
	maxBytes := fileSize * 2
	ttl := 5 * time.Minute

	cache, err := NewLRUCache(cacheDir, maxBytes, ttl)
	if err != nil {
		t.Fatalf("NewLRUCache failed: %v", err)
	}
	defer cache.Close()

	// Create and cache 2 files
	testFile1 := filepath.Join(t.TempDir(), "test1.txt")
	testData := make([]byte, fileSize)
	if err := os.WriteFile(testFile1, testData, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	if err := cache.Put("key-1", testFile1, fileSize); err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	time.Sleep(10 * time.Millisecond)

	testFile2 := filepath.Join(t.TempDir(), "test2.txt")
	if err := os.WriteFile(testFile2, testData, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	if err := cache.Put("key-2", testFile2, fileSize); err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	// Access key-1 to make it most recently used
	_, err2 := cache.Get("key-1")
	if err2 != nil {
		t.Fatalf("Get failed: %v", err2)
	}

	// Add new file - should evict key-2 (was least recently used)
	testFile3 := filepath.Join(t.TempDir(), "test3.txt")
	if err := os.WriteFile(testFile3, testData, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	if err := cache.Put("key-3", testFile3, fileSize); err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	// key-1 should still exist (was most recently used)
	if !cache.Exists("key-1") {
		t.Error("key-1 should still exist (was most recently used)")
	}

	// key-2 should be evicted (was least recently used)
	if cache.Exists("key-2") {
		t.Error("key-2 should be evicted (was least recently used)")
	}

	// key-3 should exist
	if !cache.Exists("key-3") {
		t.Error("key-3 should exist")
	}
}

// TestOversizedFile tests behavior when file size exceeds cache size.
func TestOversizedFile(t *testing.T) {
	cacheDir := t.TempDir()
	maxBytes := int64(100)
	ttl := 5 * time.Minute

	cache, err := NewLRUCache(cacheDir, maxBytes, ttl)
	if err != nil {
		t.Fatalf("NewLRUCache failed: %v", err)
	}
	defer cache.Close()

	// Try to cache a file larger than maxBytes
	testFile := filepath.Join(t.TempDir(), "large.txt")
	largeData := make([]byte, 200) // Larger than maxBytes
	if err := os.WriteFile(testFile, largeData, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	err = cache.Put("oversized-key", testFile, int64(len(largeData)))
	if err == nil {
		t.Error("Expected Put to fail for oversized file")
	}

	// Verify entry was not added
	if cache.Exists("oversized-key") {
		t.Error("Oversized entry should not be in cache")
	}
}

// BenchmarkPut benchmarks Put operation performance.
func BenchmarkPut(b *testing.B) {
	cacheDir := b.TempDir()
	maxBytes := int64(100 * 1024 * 1024) // 100MB
	ttl := 5 * time.Minute

	cache, err := NewLRUCache(cacheDir, maxBytes, ttl)
	if err != nil {
		b.Fatalf("NewLRUCache failed: %v", err)
	}
	defer cache.Close()

	testFile := filepath.Join(b.TempDir(), "bench.txt")
	testData := make([]byte, 1024) // 1KB
	if err := os.WriteFile(testFile, testData, 0644); err != nil {
		b.Fatalf("Failed to create test file: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key-%d", i)
		_ = cache.Put(key, testFile, int64(len(testData)))
	}
}

// BenchmarkGet benchmarks Get operation performance.
func BenchmarkGet(b *testing.B) {
	cacheDir := b.TempDir()
	maxBytes := int64(100 * 1024 * 1024)
	ttl := 5 * time.Minute

	cache, err := NewLRUCache(cacheDir, maxBytes, ttl)
	if err != nil {
		b.Fatalf("NewLRUCache failed: %v", err)
	}
	defer cache.Close()

	// Pre-populate cache
	testFile := filepath.Join(b.TempDir(), "bench.txt")
	testData := make([]byte, 1024)
	if err := os.WriteFile(testFile, testData, 0644); err != nil {
		b.Fatalf("Failed to create test file: %v", err)
	}

	key := "bench-key"
	if err := cache.Put(key, testFile, int64(len(testData))); err != nil {
		b.Fatalf("Put failed: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = cache.Get(key)
	}
}
