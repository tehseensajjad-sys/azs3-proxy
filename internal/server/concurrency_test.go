package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestAdaptiveConcurrencyAdjustsUpAndDown(t *testing.T) {
	l := newAdaptiveConcurrencyLimiter(1, 4, 200*time.Millisecond)

	if err := l.Acquire(context.Background()); err != nil {
		t.Fatalf("acquire failed: %v", err)
	}
	l.Release(50*time.Millisecond, true)
	if got := l.CurrentLimit(); got != 2 {
		t.Fatalf("expected limit to increase to 2, got %d", got)
	}

	if err := l.Acquire(context.Background()); err != nil {
		t.Fatalf("second acquire failed: %v", err)
	}
	l.Release(2*time.Second, true)
	if got := l.CurrentLimit(); got != 1 {
		t.Fatalf("expected limit to decrease to 1, got %d", got)
	}

	l.Release(300*time.Millisecond, false)
	if got := l.CurrentLimit(); got != 1 {
		t.Fatalf("limit should stay at min on failure, got %d", got)
	}
}

func TestConcurrencyMiddlewareSkipsHealthAndAdmin(t *testing.T) {
	s := &S3ProxyServer{concLimiter: newAdaptiveConcurrencyLimiter(1, 2, 200*time.Millisecond)}

	handler := s.concurrencyMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for _, path := range []string{"/health", "/ping", "/admin/caps"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 for %s, got %d", path, w.Code)
		}
	}
}

func TestConcurrencyMiddlewareBusyResponse(t *testing.T) {
	l := newAdaptiveConcurrencyLimiter(1, 1, 200*time.Millisecond)
	s := &S3ProxyServer{concLimiter: l, logger: zap.NewNop()}

	// Occupy the only slot
	if err := l.Acquire(context.Background()); err != nil {
		t.Fatalf("pre-acquire failed: %v", err)
	}

	handler := s.concurrencyMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/some", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 when limiter exhausted, got %d", w.Code)
	}
}
