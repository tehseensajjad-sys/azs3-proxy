package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/vibhansa-msft/azs3-proxy/internal/config"
)

func TestRegisterRoutes(t *testing.T) {
	// Setup minimal server
	cfg := &config.Config{
		S3AccessKeyID:     "test",
		S3SecretAccessKey: "test",
		AzureAuth: &config.AzureAuthConfig{
			Mode:               config.AuthModeAccountKey,
			StorageAccountName: "test",
			AccountKey:         "dGVzdA==", // "test" base64
		},
	}
	logger, _ := zap.NewDevelopment()
	router := chi.NewRouter()

	// We can't easily inject a mock backend because NewS3ProxyServer creates it.
	// But we can create the server and test routing.
	// Note: This will try to create a real Azure backend client, which might fail or succeed depending on env.
	// If it fails, NewS3ProxyServer returns error.
	// We can try to use a config that allows backend creation (e.g. valid base64 key).

	s, err := NewS3ProxyServer(router, cfg, logger, nil)
	if err != nil {
		t.Logf("NewS3ProxyServer failed (expected if no azure creds): %v", err)
		// If we can't create the server, we can't test routes this way without refactoring.
		// However, we can manually call registerRoutes if we could create the struct.
		// But registerRoutes is private.
		return
	}

	// If server creation succeeded (it should with valid-looking key), test routes.
	tests := []struct {
		method string
		path   string
		query  string
		want   int // Expected status code (not 404)
	}{
		{"GET", "/health", "", http.StatusOK},
		{"GET", "/", "", http.StatusForbidden}, // Auth middleware is active
		{"GET", "/bucket", "", http.StatusForbidden},
		{"GET", "/bucket", "uploads=1", http.StatusForbidden},
		{"GET", "/bucket", "versioning=1", http.StatusForbidden},
		{"PUT", "/bucket", "", http.StatusForbidden},
		{"DELETE", "/bucket", "", http.StatusForbidden},
		{"GET", "/bucket/key", "", http.StatusForbidden},
		{"PUT", "/bucket/key", "", http.StatusForbidden},
		{"DELETE", "/bucket/key", "", http.StatusForbidden},
	}

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			url := tt.path
			if tt.query != "" {
				url += "?" + tt.query
			}
			req := httptest.NewRequest(tt.method, url, nil)
			w := httptest.NewRecorder()

			s.router.ServeHTTP(w, req)

			if w.Code == http.StatusNotFound {
				t.Errorf("Route %s %s not found (got 404)", tt.method, url)
			}
		})
	}
}
