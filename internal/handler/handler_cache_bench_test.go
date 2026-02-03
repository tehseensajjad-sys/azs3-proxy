package handler

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/vibhansa-msft/azs3-proxy/internal/cache"
)

// Benchmark read-heavy workloads with and without local cache to compare handler overhead.
func BenchmarkGetObjectHandlerCache(b *testing.B) {
	benchmarks := []struct {
		name        string
		enableCache bool
		objectSize  int
	}{
		{"CacheDisabled_1MiB", false, 1 << 20},
		{"CacheEnabled_1MiB", true, 1 << 20},
		{"CacheDisabled_8MiB", false, 8 << 20},
		{"CacheEnabled_8MiB", true, 8 << 20},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			data := bytes.Repeat([]byte("a"), bm.objectSize)
			fb := &fakeBackend{getObjectData: data}

			h := NewS3Handler(fb, testLogger())

			// Optional cache setup
			var cleanup func()
			if bm.enableCache {
				cm, err := cache.NewCacheManager(b.TempDir(), int64(64<<20), 300)
				if err != nil {
					b.Fatalf("cache init failed: %v", err)
				}
				h.SetCacheManager(cm)
				cleanup = func() { _ = cm.Close() }
			} else {
				cleanup = func() {}
			}
			defer cleanup()

			// Seed cache (if enabled) with first request
			seedReq := newGetObjectRequest()
			seedRec := httptest.NewRecorder()
			h.GetObjectHandler(seedRec, seedReq)

			if bm.enableCache {
				ensureCached(b, h.cacheManager, "bucket", "key")
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				rec := httptest.NewRecorder()
				req := newGetObjectRequest()
				h.GetObjectHandler(rec, req)
			}
			b.StopTimer()

			// Validate backend usage to confirm cache hit behavior
			if bm.enableCache {
				if fb.getObjectCalls != 1 {
					b.Fatalf("expected 1 backend GetObject call with cache warmup, got %d", fb.getObjectCalls)
				}
			} else {
				expected := b.N + 1 // seed + each iteration
				if fb.getObjectCalls != expected {
					b.Fatalf("expected %d backend GetObject calls without cache, got %d", expected, fb.getObjectCalls)
				}
			}
		})
	}
}

// testLogger returns a quiet logger for benchmarks.
func testLogger() *zap.Logger {
	return zap.NewNop()
}

// newGetObjectRequest builds a chi-aware GET request for the handler.
func newGetObjectRequest() *http.Request {
	req := httptest.NewRequest("GET", "/bucket/key", nil)
	ctx := chi.NewRouteContext()
	ctx.URLParams.Add("bucket", "bucket")
	ctx.URLParams.Add("*", "/key")
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, ctx))
}

// ensureCached blocks briefly until the object is present in cache (or fails the benchmark).
func ensureCached(b *testing.B, cm *cache.CacheManager, bucket, key string) {
	if cm == nil {
		return
	}
	cacheKey := generateCacheKey(bucket, key)
	deadline := time.Now().Add(500 * time.Millisecond)
	for {
		if path, err := cm.GetObjectFromCache(cacheKey); err == nil && path != "" {
			return
		}
		if time.Now().After(deadline) {
			b.Fatalf("cache not populated for key %s", cacheKey)
		}
		time.Sleep(5 * time.Millisecond)
	}
}
