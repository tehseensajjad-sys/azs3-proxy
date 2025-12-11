package cache

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/vibhansa-msft/s3-azure-proxy/internal/telemetry"
)

// CacheManager manages cached objects by wrapping the LRUCache and providing
// higher-level operations for storing and retrieving S3 objects.
// It handles reading from source, writing to cache, and serving from cache.
type CacheManager struct {
	cache  *LRUCache          // Underlying LRU cache implementation
	telMgr *telemetry.Manager // Optional telemetry manager for metrics export
	ctx    context.Context    // Context for telemetry operations
}

// NewCacheManager creates a new cache manager with specified parameters.
// It initializes the underlying LRU cache with TTL support.
//
// Parameters:
//   - cacheDir: Directory path where cached files will be stored
//   - maxBytes: Maximum total size of cached files in bytes
//   - ttlSeconds: Time To Live for cache entries in seconds
//
// Returns error if the cache cannot be initialized.
func NewCacheManager(cacheDir string, maxBytes int64, ttlSeconds int) (*CacheManager, error) {
	ttl := time.Duration(ttlSeconds) * time.Second

	lruCache, err := NewLRUCache(cacheDir, maxBytes, ttl)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize LRU cache: %w", err)
	}

	return &CacheManager{
		cache:  lruCache,
		telMgr: nil,
		ctx:    context.Background(),
	}, nil
}

// SetTelemetryManager sets the telemetry manager for recording cache metrics.
// This is called during server initialization if telemetry is configured.
func (cm *CacheManager) SetTelemetryManager(tm *telemetry.Manager) {
	cm.telMgr = tm
	if cm.cache != nil && tm != nil {
		cm.cache.SetTelemetryManager(tm)
		// ensure cache also has a context for telemetry
		cm.cache.mutex.Lock()
		cm.cache.ctx = cm.ctx
		cm.cache.mutex.Unlock()
	}
}

// It writes the data to a temporary file and then moves it into the cache.
//
// Parameters:
//   - key: Unique identifier for the cached object (e.g., "bucket/key")
//   - data: Byte slice containing the complete object data
//
// Returns error if caching fails (e.g., insufficient disk space, I/O errors).
func (cm *CacheManager) CacheObject(key string, data []byte) error {
	// Create temporary file to store the object data
	tempFile, err := os.CreateTemp("", "s3-cache-*")
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}
	tempPath := tempFile.Name()
	defer func() { _ = os.Remove(tempPath) }() // Clean up temp file after caching

	// Write all data to temporary file
	if _, err := tempFile.Write(data); err != nil {
		_ = tempFile.Close()
		return fmt.Errorf("failed to write to temporary file: %w", err)
	}
	_ = tempFile.Close()

	// Move temporary file into cache
	if err := cm.cache.Put(key, tempPath, int64(len(data))); err != nil {
		return fmt.Errorf("failed to add object to cache: %w", err)
	}

	// Record cache operation in telemetry
	if cm.telMgr != nil {
		cm.telMgr.RecordCacheOperation(cm.ctx, "put")
	}

	return nil
}

// GetObjectFromCache retrieves a cached object's file path if it exists and is valid.
// Returns the file path if the object is cached and hasn't expired.
// Returns empty string if the object is not cached, has expired, or no longer exists.
func (cm *CacheManager) GetObjectFromCache(key string) (string, error) {
	filePath, err := cm.cache.Get(key)

	// Record cache hit/miss in telemetry
	if cm.telMgr != nil {
		bucket := key
		if i := strings.IndexByte(key, '/'); i >= 0 {
			bucket = key[:i]
		}
		if filePath != "" {
			cm.telMgr.RecordCacheHit(cm.ctx, bucket)
		} else {
			cm.telMgr.RecordCacheMiss(cm.ctx, bucket)
		}
	}

	return filePath, err
}

// ReadCachedObject reads the entire contents of a cached object.
// Returns the data and error (nil if successful).
// Returns error if the object is not cached or cannot be read.
func (cm *CacheManager) ReadCachedObject(key string) ([]byte, error) {
	filePath, err := cm.cache.Get(key)
	if err != nil {
		return nil, fmt.Errorf("failed to get cached object: %w", err)
	}

	if filePath == "" {
		return nil, fmt.Errorf("object not cached")
	}

	// Read entire file contents
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read cached file: %w", err)
	}

	return data, nil
}

// InvalidateObject removes an object from the cache.
// This can be called when an object is updated or deleted upstream.
func (cm *CacheManager) InvalidateObject(key string) error {
	err := cm.cache.Delete(key)

	// Record cache eviction in telemetry
	if cm.telMgr != nil && err == nil {
		cm.telMgr.RecordCacheEviction(cm.ctx, "manual")
	}

	return err
}

// InvalidateAll removes all cached objects.
// This can be called when the cache needs to be completely cleared.
func (cm *CacheManager) InvalidateAll() error {
	err := cm.cache.Clear()

	// Record cache eviction in telemetry
	if cm.telMgr != nil && err == nil {
		cm.telMgr.RecordCacheEviction(cm.ctx, "clear_all")
	}

	return err
}

// CacheSize returns the current total size of all cached objects in bytes.
func (cm *CacheManager) CacheSize() int64 {
	return cm.cache.Size()
}

// CacheCount returns the number of objects currently cached.
func (cm *CacheManager) CacheCount() int {
	return cm.cache.Count()
}

// Close cleans up cache resources and stops background cleanup processes.
// Should be called before shutting down the application.
func (cm *CacheManager) Close() error {
	return cm.cache.Close()
}

// GetStats returns cache statistics including hits, misses, evictions, and more.
func (cm *CacheManager) GetStats() Statistics {
	return cm.cache.GetStats()
}
