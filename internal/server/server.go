package server

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"

	"github.com/vibhansa-msft/s3-azure-proxy/internal/auth"
	"github.com/vibhansa-msft/s3-azure-proxy/internal/backend"
	"github.com/vibhansa-msft/s3-azure-proxy/internal/backend/azureblob"
	"github.com/vibhansa-msft/s3-azure-proxy/internal/config"
	"github.com/vibhansa-msft/s3-azure-proxy/internal/handler"
)

// S3ProxyServer represents the S3 proxy server
type S3ProxyServer struct {
	router  *chi.Mux
	config  *config.Config
	logger  *zap.Logger
	backend backend.StorageBackend
	auth    *auth.AuthVerifier
}

// NewS3ProxyServer creates and initializes a new S3 proxy server
func NewS3ProxyServer(router *chi.Mux, cfg *config.Config, logger *zap.Logger) (*S3ProxyServer, error) {
	s := &S3ProxyServer{
		router: router,
		config: cfg,
		logger: logger,
	}

	// Initialize S3 authentication verifier
	s.auth = auth.NewAuthVerifier(cfg.S3AccessKeyID, cfg.S3SecretAccessKey)

	// Initialize Azure Blob backend with flexible authentication
	backendImpl, err := azureblob.NewAzureBlobBackendWithAuth(cfg.AzureAuth, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize azure blob backend: %w", err)
	}

	s.backend = backendImpl

	// Log auth mode being used
	logger.Info("azure authentication initialized", zap.String("auth_mode", cfg.AzureAuth.Mode.String()), zap.String("storage_account", cfg.AzureAuth.StorageAccountName))

	// Add middleware
	s.registerMiddleware()

	// Register routes
	s.registerRoutes()

	return s, nil
}

// registerMiddleware registers all middleware
func (s *S3ProxyServer) registerMiddleware() {
	s.router.Use(middleware.Logger)
	s.router.Use(middleware.Recoverer)
	// Add SigV4 auth middleware
	s.router.Use(s.authMiddleware)
}

// authMiddleware validates SigV4 signatures
func (s *S3ProxyServer) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip auth for health check endpoints
		if r.URL.Path == "/health" || r.URL.Path == "/ping" {
			next.ServeHTTP(w, r)
			return
		}

		// Verify signature
		if err := s.auth.VerifySignature(r); err != nil {
			s.logger.Warn("signature verification failed", zap.Error(err))
			w.Header().Set("Content-Type", "application/xml")
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?><Error><Code>AccessDenied</Code><Message>The request signature we calculated does not match the signature you provided.</Message></Error>`))
			return
		}

		next.ServeHTTP(w, r)
	})
}

// registerRoutes registers all S3 API routes
func (s *S3ProxyServer) registerRoutes() {
	s3Handler := handler.NewS3Handler(s.backend, s.logger)

	// Health check endpoints (before auth)
	s.router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Bucket routes
	s.router.Put("/{bucket}", s3Handler.CreateBucketHandler)
	s.router.Delete("/{bucket}", s3Handler.DeleteBucketHandler)
	s.router.Get("/", s3Handler.ListBucketsHandler)

	// Object routes - must come after bucket routes to avoid conflicts
	s.router.Get("/{bucket}/*", s3Handler.ListObjectsV2Handler)
	s.router.Put("/{bucket}/*", s3Handler.PutObjectHandler)
	s.router.Get("/{bucket}/*", s3Handler.GetObjectHandler)
	s.router.Head("/{bucket}/*", s3Handler.HeadObjectHandler)
	s.router.Delete("/{bucket}/*", s3Handler.DeleteObjectHandler)

	// Multipart upload routes
	s.router.Post("/{bucket}/*", s3Handler.PostObjectHandler)
}
