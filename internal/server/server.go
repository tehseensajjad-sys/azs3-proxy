package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/vibhansa-msft/azs3-proxy/internal/auth"
	"github.com/vibhansa-msft/azs3-proxy/internal/backend"
	"github.com/vibhansa-msft/azs3-proxy/internal/backend/azureblob"
	"github.com/vibhansa-msft/azs3-proxy/internal/backend/azurefile"
	"github.com/vibhansa-msft/azs3-proxy/internal/cache"
	"github.com/vibhansa-msft/azs3-proxy/internal/config"
	"github.com/vibhansa-msft/azs3-proxy/internal/handler"
	"github.com/vibhansa-msft/azs3-proxy/internal/telemetry"
)

// S3ProxyServer represents the HTTP server that proxies S3 API requests to Azure Blob Storage.
// It handles routing, request validation, and coordinates backend storage operations.
type S3ProxyServer struct {
	router       *chi.Mux               // Chi router for HTTP request routing
	config       *config.Config         // Configuration containing auth and server settings
	logger       *zap.Logger            // Logger for request and error logging
	backend      backend.StorageBackend // Storage backend implementation (Azure Blob Storage)
	auth         *auth.AuthVerifier     // AWS SigV4 signature verifier
	cacheManager *cache.CacheManager    // Optional cache manager for local object caching
	telMgr       *telemetry.Manager     // Optional telemetry manager for metrics and logs
}

// requestIDMiddleware injects a GUID-style request ID into context and response headers.
func requestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := r.Header.Get(middleware.RequestIDHeader)
		if reqID == "" {
			reqID = uuid.NewString()
		}

		ctx := context.WithValue(r.Context(), middleware.RequestIDKey, reqID)
		w.Header().Set(middleware.RequestIDHeader, reqID)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// NewS3ProxyServer creates and initializes a new S3 proxy server with the provided configuration.
// It sets up the storage backend, authentication verifier, middleware, and routes.
// Optionally accepts a telemetry manager for metrics collection.
// Returns error if backend initialization fails.
func NewS3ProxyServer(router *chi.Mux, cfg *config.Config, logger *zap.Logger, telMgr *telemetry.Manager) (*S3ProxyServer, error) {
	s := &S3ProxyServer{
		router: router,
		config: cfg,
		logger: logger,
		telMgr: telMgr,
	}

	// Initialize the AWS SigV4 signature verifier with S3 credentials
	s.auth = auth.NewAuthVerifier(cfg.S3AccessKeyID, cfg.S3SecretAccessKey)

	// Initialize the appropriate Azure backend based on configuration
	var backendImpl backend.StorageBackend
	var err error

	switch cfg.AzureBackendType {
	case "blob":
		logger.Info("initializing Azure Blob Storage backend")
		backendImpl, err = azureblob.NewAzureBlobBackendWithAuth(cfg.AzureAuth, logger, telMgr)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize azure blob backend: %w", err)
		}
	case "file":
		logger.Info("initializing Azure Files backend")
		backendImpl, err = azurefile.NewAzureFileBackendWithAuth(cfg.AzureAuth, logger, telMgr)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize azure file backend: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported backend type: %s (must be 'blob' or 'file')", cfg.AzureBackendType)
	}

	s.backend = backendImpl

	// Log which backend and authentication method is being used for Azure
	logger.Info("azure backend initialized",
		zap.String("backend_type", cfg.AzureBackendType),
		zap.String("auth_mode", cfg.AzureAuth.Mode.String()),
		zap.String("storage_account", cfg.AzureAuth.StorageAccountName))

	// Register all HTTP middleware (logging, recovery, authentication)
	s.registerMiddleware()

	// Register all S3 API routes
	s.registerRoutes()

	return s, nil
}

// registerMiddleware registers all HTTP middleware in the correct order.
// Middleware is executed in order: RequestID -> Logger -> Recoverer -> SigV4 Auth
func (s *S3ProxyServer) registerMiddleware() {
	// Add RequestID middleware first to ensure all logs have a request ID
	s.router.Use(requestIDMiddleware)

	// Standard logging middleware for all requests
	s.router.Use(middleware.Logger)

	// Panic recovery middleware to prevent server crashes
	s.router.Use(middleware.Recoverer)

	// Custom AWS SigV4 signature verification middleware
	s.router.Use(s.authMiddleware)

	// Custom logging middleware to inject Zap logger with RequestID into context
	// and log both request start and completion (response)
	s.router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			reqID := middleware.GetReqID(r.Context())

			// Create a child logger with the request ID
			requestLogger := s.logger.With(zap.String("req_id", reqID))

			// Inject logger with request ID into context
			// We use a string key "requestID" to make it accessible without importing chi middleware in other packages
			ctx := context.WithValue(r.Context(), "requestID", reqID) //nolint:staticcheck // string key aligns with RequestIDFromContext

			requestLogger.Debug("request started",
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.String("remote_addr", r.RemoteAddr),
				zap.String("user_agent", r.UserAgent()))

			// Pass the request ID down via response header so client can trace it too
			w.Header().Set("x-amz-request-id", reqID)
			w.Header().Set("X-Request-ID", reqID)

			// Wrap ResponseWriter to capture status code
			rw := &responseCapture{ResponseWriter: w}

			next.ServeHTTP(rw, r.WithContext(ctx))

			// Log request completion
			requestLogger.Debug("request completed",
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.Int("status", rw.status),
				zap.Duration("duration", time.Since(start)))
		})
	})
	// Optional response debug middleware (enabled via DEBUG_RESPONSES env var)
	if os.Getenv("DEBUG_RESPONSES") == "true" {
		s.logger.Warn("response debug middleware enabled")
		s.router.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Wrap ResponseWriter to capture status and headers
				rw := &responseCapture{ResponseWriter: w}
				next.ServeHTTP(rw, r)
				// Only log for object/bucket paths to reduce noise
				if strings.HasPrefix(r.URL.Path, "/") {
					s.logger.Info("response debug",
						zap.String("method", r.Method),
						zap.String("path", r.URL.Path),
						zap.Int("status", rw.status),
						zap.Any("headers", rw.headers))
				}
			})
		})
	}
}

// responseCapture captures status code and headers written to a ResponseWriter
type responseCapture struct {
	http.ResponseWriter
	status  int
	headers http.Header
}

func (r *responseCapture) WriteHeader(statusCode int) {
	r.status = statusCode
	r.headers = r.ResponseWriter.Header().Clone()
	r.ResponseWriter.WriteHeader(statusCode)
}

// Write ensures we capture the default 200 status and headers when
// handlers write the body without explicitly calling WriteHeader.
func (r *responseCapture) Write(b []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
		r.headers = r.ResponseWriter.Header().Clone()
	}
	return r.ResponseWriter.Write(b)
}

// authMiddleware validates incoming requests with AWS SigV4 signature verification.
// Skips authentication for health check endpoints and logs all request details.
// Returns 403 Forbidden if signature verification fails.
func (s *S3ProxyServer) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip authentication for health check endpoints
		if r.URL.Path == "/health" || r.URL.Path == "/ping" {
			s.logger.Debug("health check request",
				zap.String("path", r.URL.Path),
				zap.String("method", r.Method))

			next.ServeHTTP(w, r)
			return
		}

		// Log request details for debugging and monitoring
		s.logger.Debug("processing request",
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
			zap.String("direction", "inbound"),
			zap.String("query", r.URL.RawQuery),
			zap.String("remote_addr", r.RemoteAddr))

		// Verify AWS SigV4 signature
		if err := s.auth.VerifySignature(r); err != nil {
			s.logger.Warn("signature verification failed",
				zap.Error(err),
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.String("remote_addr", r.RemoteAddr))

			// Return S3-formatted error response
			w.Header().Set("Content-Type", "application/xml")
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?><Error><Code>AccessDenied</Code><Message>The request signature we calculated does not match the signature you provided.</Message></Error>`))
			return
		}

		// Log successful signature verification
		s.logger.Debug("signature verified",
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path))

		next.ServeHTTP(w, r)
	})
}

// registerRoutes registers all S3 API routes with their corresponding handler functions.
// Routes are organized by HTTP method and path, with special handling for query parameters.
// If caching is configured, initializes a cache manager and attaches it to the handler.
// If telemetry is available, attaches it to the handler for metrics collection.
func (s *S3ProxyServer) registerRoutes() {
	// Create S3 API handler with backend and logger
	s3Handler := handler.NewS3Handler(s.backend, s.logger)

	// Attach telemetry manager if available
	if s.telMgr != nil {
		s3Handler.SetTelemetryManager(s.telMgr)
	}

	// Initialize cache manager if caching is enabled in configuration
	if s.config.CacheEnabled {
		cacheManager, err := cache.NewCacheManager(
			s.config.CachePath,
			s.config.CacheMaxSize,
			s.config.CacheTTL,
		)
		if err != nil {
			s.logger.Error("failed to initialize cache manager, caching disabled",
				zap.Error(err),
				zap.String("cache_path", s.config.CachePath))
		} else {
			// Store cache manager in server for cleanup on shutdown
			s.cacheManager = cacheManager
			// Attach cache manager to handler
			s3Handler.SetCacheManager(cacheManager)
			// Attach telemetry manager to cache manager if available
			if s.telMgr != nil {
				cacheManager.SetTelemetryManager(s.telMgr)
			}
			s.logger.Info("cache manager initialized",
				zap.String("cache_path", s.config.CachePath),
				zap.Int64("cache_max_size", s.config.CacheMaxSize),
				zap.Int("cache_ttl_seconds", s.config.CacheTTL))
		}
	}

	// Health check endpoint (no authentication required)
	s.router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	// List all buckets
	s.router.Get("/", s3Handler.ListBucketsHandler)

	// Bucket operations: GET with query parameters
	s.router.Get("/{bucket}", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("uploads") != "" {
			// List multipart uploads in progress
			s3Handler.ListMultipartUploadsHandler(w, r)
		} else if r.URL.Query().Get("versioning") != "" {
			// Get bucket versioning configuration
			s3Handler.GetVersioningHandler(w, r)
		} else if r.URL.Query().Get("versions") != "" {
			// List all object versions in bucket
			s3Handler.ListObjectVersionsHandler(w, r)
		} else {
			// List objects in bucket (default)
			s3Handler.ListObjectsV2Handler(w, r)
		}
	})

	// Bucket operations: PUT for creation and configuration
	s.router.Put("/{bucket}", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("versioning") != "" {
			// Enable versioning on bucket
			s3Handler.EnableVersioningHandler(w, r)
		} else {
			// Create new bucket
			s3Handler.CreateBucketHandler(w, r)
		}
	})

	// Delete bucket
	s.router.Delete("/{bucket}", s3Handler.DeleteBucketHandler)

	// Head bucket
	s.router.Head("/{bucket}", s3Handler.HeadBucketHandler)

	// Bucket operations: POST (DeleteObjects)
	s.router.Post("/{bucket}", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Has("delete") {
			s3Handler.DeleteObjectsHandler(w, r)
		} else {
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	// Object operations: GET with query parameters
	// Must come after bucket routes to avoid path conflicts
	s.router.Get("/{bucket}/*", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("uploadId") != "" {
			// List parts of an active multipart upload
			s3Handler.ListPartsHandler(w, r)
		} else if r.URL.Query().Get("versionId") != "" {
			// Get specific version of an object
			s3Handler.GetObjectVersionHandler(w, r)
		} else {
			// Get object (current version)
			s3Handler.GetObjectHandler(w, r)
		}
	})

	// Object operations: PUT for uploads and multipart
	s.router.Put("/{bucket}/*", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("uploadId") != "" && r.URL.Query().Get("partNumber") != "" {
			// Upload a part in a multipart upload
			s3Handler.UploadPartHandler(w, r)
		} else {
			// Put object (single upload)
			s3Handler.PutObjectHandler(w, r)
		}
	})

	// Object operations: HEAD for metadata
	s.router.Head("/{bucket}/*", s3Handler.HeadObjectHandler)

	// Object operations: DELETE with query parameters
	s.router.Delete("/{bucket}/*", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("uploadId") != "" {
			// Abort an in-progress multipart upload
			s3Handler.AbortMultipartUploadHandler(w, r)
		} else if r.URL.Query().Get("versionId") != "" {
			// Delete specific version of an object
			s3Handler.DeleteObjectVersionHandler(w, r)
		} else {
			// Delete object (add delete marker if versioning enabled)
			s3Handler.DeleteObjectHandler(w, r)
		}
	})

	// Object operations: POST for multipart upload completion
	s.router.Post("/{bucket}/*", s3Handler.PostObjectHandler)
}

// Close performs graceful shutdown of the server resources.
// It closes the cache manager if it was initialized.
// Should be called before application shutdown.
func (s *S3ProxyServer) Close() error {
	if s.backend != nil {
		if err := s.backend.Close(); err != nil {
			s.logger.Error("failed to close backend", zap.Error(err))
			// Continue closing other resources
		}
	}

	if s.cacheManager != nil {
		if err := s.cacheManager.Close(); err != nil {
			s.logger.Error("failed to close cache manager", zap.Error(err))
			return err
		}
	}
	return nil
}
