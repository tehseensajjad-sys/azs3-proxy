package cache

import (
	"os"
	"testing"
)

func TestNewCacheManager(t *testing.T) {
	tmpDir := t.TempDir()
	// defer os.RemoveAll(tmpDir) - t.TempDir handles cleanup

	cm, err := NewCacheManager(tmpDir, 10*1024*1024, 3600)
	if err != nil {
		t.Fatalf("NewCacheManager failed: %v", err)
	}
	defer func() { _ = cm.Close() }()

	if cm == nil {
		t.Fatal("expected non-nil cache manager")
	}
}

func TestCacheObject(t *testing.T) {
	tmpDir := t.TempDir()

	cm, err := NewCacheManager(tmpDir, 10*1024*1024, 3600)
	if err != nil {
		t.Fatalf("NewCacheManager failed: %v", err)
	}
	defer func() { _ = cm.Close() }()

	testData := []byte("test data for caching")
	cacheKey := "test-bucket/test-key"

	err = cm.CacheObject(cacheKey, testData)
	if err != nil {
		t.Fatalf("CacheObject failed: %v", err)
	}

	// Verify the file was created
	files, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatalf("ReadDir failed: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("expected cached file to be created")
	}
}

func TestGetObjectFromCache(t *testing.T) {
	tmpDir := t.TempDir()

	cm, err := NewCacheManager(tmpDir, 10*1024*1024, 3600)
	if err != nil {
		t.Fatalf("NewCacheManager failed: %v", err)
	}
	defer func() { _ = cm.Close() }()

	testData := []byte("cached object data")
	cacheKey := "bucket1/object1"

	// Cache an object
	err = cm.CacheObject(cacheKey, testData)
	if err != nil {
		t.Fatalf("CacheObject failed: %v", err)
	}

	// Retrieve from cache
	filePath, err := cm.GetObjectFromCache(cacheKey)
	if err != nil {
		t.Fatalf("GetObjectFromCache failed: %v", err)
	}
	if filePath == "" {
		t.Fatal("expected non-empty file path")
	}

	// Verify the content
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if string(data) != string(testData) {
		t.Errorf("expected %s, got %s", testData, data)
	}
}

func TestReadCachedObject(t *testing.T) {
	tmpDir := t.TempDir()
	defer func() { _ = os.RemoveAll(tmpDir) }()

	cm, err := NewCacheManager(tmpDir, 10*1024*1024, 3600)
	if err != nil {
		t.Fatalf("NewCacheManager failed: %v", err)
	}
	defer func() { _ = cm.Close() }()

	testData := []byte("test object content")
	cacheKey := "bucket2/object2"

	// Cache an object
	err = cm.CacheObject(cacheKey, testData)
	if err != nil {
		t.Fatalf("CacheObject failed: %v", err)
	}

	// Read from cache
	data, err := cm.ReadCachedObject(cacheKey)
	if err != nil {
		t.Fatalf("ReadCachedObject failed: %v", err)
	}

	if string(data) != string(testData) {
		t.Errorf("expected %s, got %s", testData, data)
	}
}

func TestInvalidateObject(t *testing.T) {
	tmpDir := t.TempDir()
	defer func() { _ = os.RemoveAll(tmpDir) }()

	cm, err := NewCacheManager(tmpDir, 10*1024*1024, 3600)
	if err != nil {
		t.Fatalf("NewCacheManager failed: %v", err)
	}
	defer func() { _ = cm.Close() }()

	testData := []byte("data to invalidate")
	cacheKey := "bucket3/object3"

	// Cache an object
	err = cm.CacheObject(cacheKey, testData)
	if err != nil {
		t.Fatalf("CacheObject failed: %v", err)
	}

	// Verify it's in cache
	filePath, err := cm.GetObjectFromCache(cacheKey)
	if err != nil || filePath == "" {
		t.Fatal("expected object in cache before invalidation")
	}

	// Invalidate it
	err = cm.InvalidateObject(cacheKey)
	if err != nil {
		t.Fatalf("InvalidateObject failed: %v", err)
	}

	// Try to read from cache - should fail now
	_, err = cm.ReadCachedObject(cacheKey)
	if err == nil {
		t.Fatal("expected error after invalidation")
	}
}

func TestInvalidateAll(t *testing.T) {
	tmpDir := t.TempDir()
	defer func() { _ = os.RemoveAll(tmpDir) }()

	cm, err := NewCacheManager(tmpDir, 10*1024*1024, 3600)
	if err != nil {
		t.Fatalf("NewCacheManager failed: %v", err)
	}
	defer func() { _ = cm.Close() }()

	// Cache multiple objects
	for i := 1; i <= 3; i++ {
		key := "bucket/object" + string(rune('0'+i))
		data := []byte("test data " + string(rune('0'+i)))
		err := cm.CacheObject(key, data)
		if err != nil {
			t.Fatalf("CacheObject failed: %v", err)
		}
	}

	initialCount := cm.CacheCount()
	if initialCount != 3 {
		t.Errorf("expected 3 cached objects, got %d", initialCount)
	}

	// Invalidate all
	_ = cm.InvalidateAll()

	// Verify all are gone
	finalCount := cm.CacheCount()
	if finalCount != 0 {
		t.Errorf("expected 0 cached objects after InvalidateAll, got %d", finalCount)
	}
}

func TestCacheSize(t *testing.T) {
	tmpDir := t.TempDir()
	defer func() { _ = os.RemoveAll(tmpDir) }()

	cm, err := NewCacheManager(tmpDir, 10*1024*1024, 3600)
	if err != nil {
		t.Fatalf("NewCacheManager failed: %v", err)
	}
	defer func() { _ = cm.Close() }()

	testData := []byte("test data with known size")
	cacheKey := "bucket/key"

	initialSize := cm.CacheSize()

	err = cm.CacheObject(cacheKey, testData)
	if err != nil {
		t.Fatalf("CacheObject failed: %v", err)
	}

	finalSize := cm.CacheSize()
	if finalSize <= initialSize {
		t.Errorf("cache size should increase after caching object")
	}
}

func TestCacheCount(t *testing.T) {
	tmpDir := t.TempDir()
	defer func() { _ = os.RemoveAll(tmpDir) }()

	cm, err := NewCacheManager(tmpDir, 10*1024*1024, 3600)
	if err != nil {
		t.Fatalf("NewCacheManager failed: %v", err)
	}
	defer func() { _ = cm.Close() }()

	if cm.CacheCount() != 0 {
		t.Errorf("expected initial count of 0, got %d", cm.CacheCount())
	}

	// Cache some objects
	for i := 1; i <= 5; i++ {
		key := "bucket/obj" + string(rune('0'+i))
		data := []byte("data " + string(rune('0'+i)))
		err := cm.CacheObject(key, data)
		if err != nil {
			t.Fatalf("CacheObject failed: %v", err)
		}
	}

	if cm.CacheCount() != 5 {
		t.Errorf("expected count of 5, got %d", cm.CacheCount())
	}
}

func TestCacheManagerClose(t *testing.T) {
	tmpDir := t.TempDir()
	defer func() { _ = os.RemoveAll(tmpDir) }()

	cm, err := NewCacheManager(tmpDir, 10*1024*1024, 3600)
	if err != nil {
		t.Fatalf("NewCacheManager failed: %v", err)
	}

	// Close should not panic
	_ = cm.Close()

	// Close again should not panic
	_ = cm.Close()
}

func TestGetStats(t *testing.T) {
	tmpDir := t.TempDir()
	defer func() { _ = os.RemoveAll(tmpDir) }()

	cm, err := NewCacheManager(tmpDir, 10*1024*1024, 3600)
	if err != nil {
		t.Fatalf("NewCacheManager failed: %v", err)
	}
	defer func() { _ = cm.Close() }()

	stats := cm.GetStats()
	// Stats should be a valid Statistics struct - at least TotalRequests should exist
	if stats.TotalRequests < 0 {
		t.Fatal("expected valid stats")
	}
}

// TestCacheObjectWithLargeData tests caching with larger data
func TestCacheObjectWithLargeData(t *testing.T) {
	tmpDir := t.TempDir()
	defer func() { _ = os.RemoveAll(tmpDir) }()

	cm, err := NewCacheManager(tmpDir, 100*1024*1024, 3600)
	if err != nil {
		t.Fatalf("NewCacheManager failed: %v", err)
	}
	defer func() { _ = cm.Close() }()

	// Create 1MB of test data
	largeData := make([]byte, 1024*1024)
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}

	cacheKey := "bucket/largefile"
	err = cm.CacheObject(cacheKey, largeData)
	if err != nil {
		t.Fatalf("CacheObject with large data failed: %v", err)
	}

	// Retrieve and verify
	filePath, err := cm.GetObjectFromCache(cacheKey)
	if err != nil {
		t.Fatalf("GetObjectFromCache failed: %v", err)
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if len(data) != len(largeData) {
		t.Errorf("expected %d bytes, got %d", len(largeData), len(data))
	}
}

// TestCacheObjectWithSpecialChars tests caching with special character keys
func TestCacheObjectWithSpecialChars(t *testing.T) {
	tmpDir := t.TempDir()
	defer func() { _ = os.RemoveAll(tmpDir) }()

	cm, err := NewCacheManager(tmpDir, 10*1024*1024, 3600)
	if err != nil {
		t.Fatalf("NewCacheManager failed: %v", err)
	}
	defer func() { _ = cm.Close() }()

	testCases := []struct {
		name     string
		cacheKey string
		data     []byte
	}{
		{
			name:     "key with spaces",
			cacheKey: "bucket/my object key",
			data:     []byte("data with spaces"),
		},
		{
			name:     "key with special chars",
			cacheKey: "bucket/object-@#$%",
			data:     []byte("data with special chars"),
		},
		{
			name:     "key with subdirs",
			cacheKey: "bucket/dir1/dir2/object",
			data:     []byte("data in subdirs"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := cm.CacheObject(tc.cacheKey, tc.data)
			if err != nil {
				t.Fatalf("CacheObject failed: %v", err)
			}

			filePath, err := cm.GetObjectFromCache(tc.cacheKey)
			if err != nil {
				t.Fatalf("GetObjectFromCache failed: %v", err)
			}
			if filePath == "" {
				t.Fatal("expected non-empty file path")
			}

			data, err := os.ReadFile(filePath)
			if err != nil {
				t.Fatalf("ReadFile failed: %v", err)
			}
			if string(data) != string(tc.data) {
				t.Errorf("expected %s, got %s", tc.data, data)
			}
		})
	}
}

// TestCacheManagerConcurrentAccess tests concurrent cache operations
func TestCacheManagerConcurrentAccess(t *testing.T) {
	tmpDir := t.TempDir()
	defer func() { _ = os.RemoveAll(tmpDir) }()

	cm, err := NewCacheManager(tmpDir, 50*1024*1024, 3600)
	if err != nil {
		t.Fatalf("NewCacheManager failed: %v", err)
	}
	defer func() { _ = cm.Close() }()

	done := make(chan bool, 10)

	// Concurrent writes
	for i := 0; i < 5; i++ {
		go func(idx int) {
			key := "bucket/obj" + string(rune('0'+idx))
			data := []byte("concurrent data " + string(rune('0'+idx)))
			_ = cm.CacheObject(key, data)
			done <- true
		}(i)
	}

	// Concurrent reads
	for i := 0; i < 5; i++ {
		go func(idx int) {
			key := "bucket/obj" + string(rune('0'+idx))
			_, _ = cm.GetObjectFromCache(key)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

// TestReadCachedObjectNonExistent tests reading non-existent cached object
func TestReadCachedObjectNonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	defer func() { _ = os.RemoveAll(tmpDir) }()

	cm, err := NewCacheManager(tmpDir, 10*1024*1024, 3600)
	if err != nil {
		t.Fatalf("NewCacheManager failed: %v", err)
	}
	defer func() { _ = cm.Close() }()

	_, err = cm.ReadCachedObject("nonexistent/key")
	if err == nil {
		t.Fatal("expected error for non-existent object")
	}
}

// TestGetObjectFromCacheNonExistent tests getting non-existent cached object
func TestGetObjectFromCacheNonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	defer func() { _ = os.RemoveAll(tmpDir) }()

	cm, err := NewCacheManager(tmpDir, 10*1024*1024, 3600)
	if err != nil {
		t.Fatalf("NewCacheManager failed: %v", err)
	}
	defer func() { _ = cm.Close() }()

	filePath, _ := cm.GetObjectFromCache("nonexistent/key")
	// Should return empty string without error for non-existent key
	if filePath != "" {
		t.Errorf("expected empty file path, got %s", filePath)
	}
}
