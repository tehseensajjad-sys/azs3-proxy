package cache

import (
	"container/list"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/vibhansa-msft/s3-azure-proxy/internal/telemetry"
)

// CacheEntry represents a single cached item with metadata.
// It tracks when the item was last accessed for LRU eviction and when it expires.
type CacheEntry struct {
	Key        string    // Unique identifier for the cache entry (e.g., bucket/key)
	FilePath   string    // Path to the file on disk
	FileSize   int64     // Size of the cached file in bytes
	CreatedAt  time.Time // When the entry was created
	LastUsedAt time.Time // When the entry was last accessed (for LRU)
	ExpiresAt  time.Time // When the entry should be considered stale
}

// LRUCache implements a Least Recently Used cache with TTL (Time To Live) support.
// It maintains cached files on disk and evicts them based on:
//  1. LRU policy: Least recently used items are evicted first
//  2. TTL/Timeout: Items older than TTL are automatically expired
//  3. Size limit: When total size exceeds maxBytes, oldest items are evicted
//
// Thread-safe for concurrent access from multiple goroutines.
type LRUCache struct {
	maxBytes      int64                    // Maximum total size of cached files in bytes
	currentBytes  int64                    // Current total size of all cached files
	cacheDir      string                   // Directory where cached files are stored
	ttl           time.Duration            // Time To Live for cache entries
	entries       map[string]*list.Element // Map of key -> linked list element
	lruList       *list.List               // Doubly-linked list for LRU ordering
	mutex         sync.RWMutex             // Protects all fields above
	cleanupTicker *time.Ticker             // Periodically cleans up expired entries
	stopCleanup   chan struct{}            // Signal to stop cleanup goroutine
	stats         *CacheStats              // Performance and operational statistics
	telMgr        *telemetry.Manager       // Optional telemetry manager for metrics
	ctx           context.Context          // Context for telemetry operations
}

// NewLRUCache creates a new LRU cache instance with specified parameters.
// It initializes the cache directory and starts a background goroutine for TTL cleanup.
//
// Parameters:
//   - cacheDir: Directory path where cached files will be stored
//   - maxBytes: Maximum total size of cached files in bytes
//   - ttl: Time To Live duration for cache entries (how long before they expire)
//
// Returns error if the cache directory cannot be created.
func NewLRUCache(cacheDir string, maxBytes int64, ttl time.Duration) (*LRUCache, error) {
	// Create cache directory if it doesn't exist
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Clean up any existing cache files from previous runs
	// This ensures we start fresh with a clean cache state
	if err := os.RemoveAll(cacheDir); err != nil {
		return nil, fmt.Errorf("failed to clean cache directory: %w", err)
	}

	// Recreate the directory after cleaning
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	cache := &LRUCache{
		maxBytes:     maxBytes,
		currentBytes: 0,
		cacheDir:     cacheDir,
		ttl:          ttl,
		entries:      make(map[string]*list.Element),
		lruList:      list.New(),
		stopCleanup:  make(chan struct{}),
		stats:        &CacheStats{},
		telMgr:       nil,
		ctx:          context.Background(),
	}

	// Start background goroutine to clean up expired entries periodically
	cache.cleanupTicker = time.NewTicker(ttl / 2)
	go cache.cleanupExpiredEntries()

	return cache, nil
}

// SetTelemetryManager sets a telemetry manager for recording LRU events.
func (lc *LRUCache) SetTelemetryManager(tm *telemetry.Manager) {
	lc.mutex.Lock()
	defer lc.mutex.Unlock()
	lc.telMgr = tm
}

// Get retrieves a cached file, returning the file path if it exists and hasn't expired.
// Updates the LRU ordering by moving the accessed item to the end (most recently used).
// Returns empty string if the entry doesn't exist or has expired.
func (lc *LRUCache) Get(key string) (string, error) {
	lc.mutex.Lock()
	defer lc.mutex.Unlock()

	lc.stats.RecordGet()

	// Look up entry in the cache
	elem, exists := lc.entries[key]
	if !exists {
		lc.stats.RecordMiss()
		if lc.telMgr != nil {
			// Reduce cardinality: extract bucket from "bucket/key" cache key
			bucket := key
			if idx := strings.IndexByte(key, '/'); idx >= 0 {
				bucket = key[:idx]
			}
			lc.telMgr.RecordCacheMiss(lc.ctx, bucket)
		}
		return "", nil // Entry not found
	}

	entry := elem.Value.(*CacheEntry)

	// Check if entry has expired based on TTL
	if time.Now().After(entry.ExpiresAt) {
		// Entry has expired, remove it and return nil
		lc.stats.RecordExpiration(entry.FileSize)
		if lc.telMgr != nil {
			lc.telMgr.RecordCacheExpiration(lc.ctx)
		}
		lc.removeEntry(elem)
		lc.stats.RecordMiss()
		if lc.telMgr != nil {
			bucket := key
			if idx := strings.IndexByte(key, '/'); idx >= 0 {
				bucket = key[:idx]
			}
			lc.telMgr.RecordCacheMiss(lc.ctx, bucket)
		}
		return "", nil
	}

	// Check if file still exists on disk
	if _, err := os.Stat(entry.FilePath); err != nil {
		// File doesn't exist, remove entry and return nil (miss)
		lc.removeEntry(elem)
		lc.stats.RecordMiss()
		if lc.telMgr != nil {
			lc.telMgr.RecordCacheMiss(lc.ctx, key)
		}
		return "", nil
	}

	// Update last used timestamp and move to end of list (most recently used)
	entry.LastUsedAt = time.Now()
	lc.lruList.MoveToBack(elem)
	lc.stats.RecordHit()
	if lc.telMgr != nil {
		bucket := key
		if idx := strings.IndexByte(key, '/'); idx >= 0 {
			bucket = key[:idx]
		}
		lc.telMgr.RecordCacheHit(lc.ctx, bucket)
	}

	return entry.FilePath, nil
}

// Put stores a file in the cache, managing LRU eviction if necessary.
// If the cache exceeds maxBytes after adding this entry, older entries are evicted.
// The filePath parameter should be the location of the file to cache.
// Returns error if the file cannot be accessed or cache operations fail.
func (lc *LRUCache) Put(key string, filePath string, fileSize int64) error {
	lc.mutex.Lock()
	defer lc.mutex.Unlock()

	// Verify the file exists and get its actual size
	info, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}

	actualSize := info.Size()

	// If file is larger than entire cache, reject it
	if actualSize > lc.maxBytes {
		return fmt.Errorf("file size (%d) exceeds cache size (%d)", actualSize, lc.maxBytes)
	}

	// If entry already exists, remove it first
	if elem, exists := lc.entries[key]; exists {
		lc.removeEntry(elem)
	}

	// Copy file to cache directory
	cacheFilePath := filepath.Join(lc.cacheDir, escapeName(key))
	if err := copyFile(filePath, cacheFilePath); err != nil {
		return fmt.Errorf("failed to copy file to cache: %w", err)
	}

	// Create new cache entry
	entry := &CacheEntry{
		Key:        key,
		FilePath:   cacheFilePath,
		FileSize:   actualSize,
		CreatedAt:  time.Now(),
		LastUsedAt: time.Now(),
		ExpiresAt:  time.Now().Add(lc.ttl),
	}

	// Add entry to linked list and map
	elem := lc.lruList.PushBack(entry)
	lc.entries[key] = elem
	lc.currentBytes += actualSize
	lc.stats.RecordPut(actualSize)

	// Evict LRU entries if cache exceeds maxBytes
	for lc.currentBytes > lc.maxBytes && lc.lruList.Len() > 0 {
		oldest := lc.lruList.Front()
		oldestEntry := oldest.Value.(*CacheEntry)
		lc.stats.RecordEviction(oldestEntry.FileSize)
		if lc.telMgr != nil {
			lc.telMgr.RecordCacheEviction(lc.ctx, "size_limit")
		}
		lc.removeEntry(oldest)
	}

	return nil
}

// Delete removes a specific cache entry and its associated file.
// Returns error if the file cannot be deleted.
func (lc *LRUCache) Delete(key string) error {
	lc.mutex.Lock()
	defer lc.mutex.Unlock()

	elem, exists := lc.entries[key]
	if !exists {
		return nil // Entry doesn't exist, nothing to do
	}

	entry := elem.Value.(*CacheEntry)

	// Delete file from disk
	if err := os.Remove(entry.FilePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete cache file: %w", err)
	}

	// Remove from cache structures
	lc.removeEntry(elem)
	return nil
}

// Clear removes all entries from the cache and deletes all cached files.
// Returns error if any files cannot be deleted.
func (lc *LRUCache) Clear() error {
	lc.mutex.Lock()
	defer lc.mutex.Unlock()

	// Delete all cached files
	for key, elem := range lc.entries {
		entry := elem.Value.(*CacheEntry)
		if err := os.Remove(entry.FilePath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to delete cache file for %s: %w", key, err)
		}
	}

	// Clear all data structures
	lc.entries = make(map[string]*list.Element)
	lc.lruList = list.New()
	lc.currentBytes = 0

	return nil
}

// Size returns the current total size of all cached files in bytes.
func (lc *LRUCache) Size() int64 {
	lc.mutex.RLock()
	defer lc.mutex.RUnlock()
	return lc.currentBytes
}

// Count returns the number of entries currently in the cache.
func (lc *LRUCache) Count() int {
	lc.mutex.RLock()
	defer lc.mutex.RUnlock()
	return len(lc.entries)
}

// Exists checks if a key exists in the cache and hasn't expired.
// Returns true only if the entry exists, is not expired, and file exists on disk.
func (lc *LRUCache) Exists(key string) bool {
	lc.mutex.RLock()
	defer lc.mutex.RUnlock()

	elem, exists := lc.entries[key]
	if !exists {
		return false
	}

	entry := elem.Value.(*CacheEntry)

	// Check if expired
	if time.Now().After(entry.ExpiresAt) {
		return false
	}

	// Check if file exists
	_, err := os.Stat(entry.FilePath)
	return err == nil
}

// Close cleans up cache resources and stops background cleanup goroutine.
// Should be called before shutting down the application.
func (lc *LRUCache) Close() error {
	// Stop the cleanup ticker
	if lc.cleanupTicker != nil {
		lc.cleanupTicker.Stop()
	}

	// Signal cleanup goroutine to stop
	select {
	case lc.stopCleanup <- struct{}{}:
	default:
	}

	return nil
}

// GetStats returns a snapshot of current cache statistics.
func (lc *LRUCache) GetStats() Statistics {
	return lc.stats.GetStats()
}

// removeEntry is a helper function that removes an entry from all cache structures.
// Must be called with the mutex already held.
func (lc *LRUCache) removeEntry(elem *list.Element) {
	entry := elem.Value.(*CacheEntry)

	// Remove from linked list
	lc.lruList.Remove(elem)

	// Remove from map
	delete(lc.entries, entry.Key)

	// Update current size
	lc.currentBytes -= entry.FileSize

	// Delete file from disk (ignore errors)
	_ = os.Remove(entry.FilePath)
	// Record eviction telemetry for explicit removals
	if lc.telMgr != nil {
		lc.telMgr.RecordCacheEviction(lc.ctx, "removed")
	}
}

// cleanupExpiredEntries runs periodically in the background to remove expired entries.
// This ensures that even if entries are never accessed again, they are eventually cleaned up.
func (lc *LRUCache) cleanupExpiredEntries() {
	for {
		select {
		case <-lc.cleanupTicker.C:
			// Periodically clean up expired entries
			lc.mutex.Lock()

			now := time.Now()
			var toRemove []*list.Element

			// Find all expired entries
			for elem := lc.lruList.Front(); elem != nil; elem = elem.Next() {
				entry := elem.Value.(*CacheEntry)
				if now.After(entry.ExpiresAt) {
					toRemove = append(toRemove, elem)
				}
			}

			// Remove expired entries
			for _, elem := range toRemove {
				entry := elem.Value.(*CacheEntry)
				lc.stats.RecordExpiration(entry.FileSize)
				if lc.telMgr != nil {
					lc.telMgr.RecordCacheExpiration(lc.ctx)
				}
				lc.removeEntry(elem)
			}

			lc.mutex.Unlock()

		case <-lc.stopCleanup:
			// Stop the cleanup goroutine
			return
		}
	}
}

// escapeName converts a key (e.g., "bucket/key/path") into a safe filename.
// This prevents directory traversal attacks and handles special characters.
func escapeName(key string) string {
	// Replace path separators and other special characters with underscores
	safe := ""
	for _, c := range key {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '.' {
			safe += string(c)
		} else {
			safe += "_"
		}
	}
	return safe
}

// copyFile copies a file from src to dst, creating dst if it doesn't exist.
// Returns error if the copy operation fails.
func copyFile(src, dst string) error {
	// Read source file
	data, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("failed to read source file: %w", err)
	}

	// Write to destination file
	if err := os.WriteFile(dst, data, 0644); err != nil {
		return fmt.Errorf("failed to write destination file: %w", err)
	}

	return nil
}
