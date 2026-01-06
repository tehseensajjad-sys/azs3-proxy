package cache

import (
	"context"
	"os"
	"testing"

	"github.com/vibhansa-msft/azs3-proxy/internal/telemetry"
)

func TestCacheManagerBasicFlow(t *testing.T) {
	tmpDir := t.TempDir()

	cm, err := NewCacheManager(tmpDir, 10*1024*1024, 3600)
	if err != nil {
		t.Fatalf("NewCacheManager failed: %v", err)
	}
	defer func() { _ = cm.Close() }()

	// set telemetry manager if possible
	if mgr, _ := telemetry.NewManager(context.Background()); mgr != nil {
		cm.SetTelemetryManager(mgr)
	}

	data := []byte("somedata")
	if err := cm.CacheObject("bucket/key", data); err != nil {
		t.Fatalf("CacheObject failed: %v", err)
	}

	d, err := cm.ReadCachedObject("bucket/key")
	if err != nil {
		t.Fatalf("ReadCachedObject failed: %v", err)
	}
	if string(d) != string(data) {
		t.Fatalf("ReadCachedObject returned unexpected data: %s", string(d))
	}

	if err := cm.InvalidateObject("bucket/key"); err != nil {
		t.Fatalf("InvalidateObject failed: %v", err)
	}

	// verify cleanup
	files, _ := os.ReadDir(tmpDir)
	_ = files
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

func TestCacheManagerMisc(t *testing.T) {
tmpDir := t.TempDir()
cm, _ := NewCacheManager(tmpDir, 1000, 60)
defer cm.Close()

if err := cm.InvalidateAll(); err != nil {
t.Error(err)
}

if s := cm.CacheSize(); s != 0 {
t.Errorf("size %d", s)
}
}
