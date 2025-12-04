package server

import (
	"context"
	"fmt"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/vibhansa-msft/s3-azure-proxy/internal/backend"
	"github.com/vibhansa-msft/s3-azure-proxy/internal/config"
	"github.com/vibhansa-msft/s3-azure-proxy/internal/handler"
)

// S3ProxyServer represents the S3 proxy server
type S3ProxyServer struct {
	router  *chi.Mux
	config  *config.Config
	logger  *zap.Logger
	backend backend.StorageBackend
}

// NewS3ProxyServer creates and initializes a new S3 proxy server
func NewS3ProxyServer(router *chi.Mux, cfg *config.Config, logger *zap.Logger) (*S3ProxyServer, error) {
	s := &S3ProxyServer{
		router: router,
		config: cfg,
		logger: logger,
	}

	// Initialize backend (Azure Blob for now)
	// This will be implemented in the Azure backend package
	backendImpl, err := NewAzureBlobBackend(cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize azure blob backend: %w", err)
	}

	s.backend = backendImpl

	// Register routes
	s.registerRoutes()

	return s, nil
}

// registerRoutes registers all S3 API routes
func (s *S3ProxyServer) registerRoutes() {
	s3Handler := handler.NewS3Handler(s.backend, s.logger)

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

// NewAzureBlobBackend creates a new Azure Blob backend (placeholder)
func NewAzureBlobBackend(cfg *config.Config, logger *zap.Logger) (backend.StorageBackend, error) {
	// TODO: Implement Azure Blob backend initialization
	return nil, fmt.Errorf("azure blob backend not yet implemented")
}
