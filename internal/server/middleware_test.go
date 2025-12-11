package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/vibhansa-msft/s3-azure-proxy/internal/auth"
	"github.com/vibhansa-msft/s3-azure-proxy/internal/config"
)

func TestAuthMiddleware(t *testing.T) {
	// Setup
	cfg := &config.Config{
		S3AccessKeyID:     "test-access",
		S3SecretAccessKey: "test-secret",
	}
	logger, _ := zap.NewDevelopment()
	verifier := auth.NewAuthVerifier(cfg.S3AccessKeyID, cfg.S3SecretAccessKey)

	s := &S3ProxyServer{
		router: chi.NewRouter(),
		config: cfg,
		logger: logger,
		auth:   verifier,
	}

	// Define a protected handler
	protectedHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("success"))
	})

	// Apply middleware
	handler := s.authMiddleware(protectedHandler)

	t.Run("HealthCheck_Skipped", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200 OK for health check, got %d", w.Code)
		}
	})

	t.Run("Ping_Skipped", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/ping", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200 OK for ping, got %d", w.Code)
		}
	})

	t.Run("Protected_NoAuth_Forbidden", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/protected", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("Expected 403 Forbidden for unauthenticated request, got %d", w.Code)
		}
	})

	// Note: Testing a valid signed request here would require generating a valid SigV4 signature.
	// That is covered in internal/auth/sigv4_test.go.
	// Here we just verify the middleware integration.
}
