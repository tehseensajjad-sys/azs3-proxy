CORE_PACKAGES.md# Core Package Implementation Guide

This document outlines all the internal packages that need to be created alongside the existing config and backend packages.

## Package Structure to Create

### 1. `internal/server/` - HTTP Server Setup
**File**: `internal/server/server.go`

```go
package server

import (
	"context"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
	"github.com/vibhansa-msft/s3-azure-proxy/internal/config"
	"github.com/vibhansa-msft/s3-azure-proxy/internal/backend"
)

type S3ProxyServer struct {
	router  *chi.Mux
	config  *config.Config
	logger  *zap.Logger
	backend backend.StorageBackend
}

func NewS3ProxyServer(router *chi.Mux, cfg *config.Config, logger *zap.Logger) (*S3ProxyServer, error) {
	// Initialize backend (Azure Blob)
	// Register routes
	// Return server
	return nil, nil
}
```

### 2. `internal/auth/` - SigV4 Authentication
**File**: `internal/auth/sigv4.go`

```go
package auth

import (
	"net/http"
	"go.uber.org/zap"
)

type AuthVerifier struct {
	accessKey string
	secretKey string
	logger    *zap.Logger
}

func (av *AuthVerifier) VerifyRequest(r *http.Request) error {
	// Extract Authorization header
	// Validate SigV4 signature
	// Return nil if valid, error otherwise
	return nil
}
```

### 3. `internal/handler/` - S3 HTTP Handlers
**Files**:
- `internal/handler/s3.go` - Main S3 handler setup
- `internal/handler/bucket.go` - Bucket operations (CreateBucket, DeleteBucket, ListBuckets, ListObjectsV2)
- `internal/handler/object.go` - Object operations (PutObject, GetObject, HeadObject, DeleteObject)
- `internal/handler/multipart.go` - Multipart upload operations
- `internal/handler/errors.go` - S3 error response formatting

### 4. `internal/backend/azureblob/` - Azure Blob Implementation
**File**: `internal/backend/azureblob/client.go`

```go
package azureblob

import (
	"context"
	"io"
	"github.com/azure/azure-sdk-for-go/sdk/storage/azblob"
	"go.uber.org/zap"
	"github.com/vibhansa-msft/s3-azure-proxy/internal/backend"
)

type AzureBlobBackend struct {
	client    *azblob.Client
	accountKey string
	logger    *zap.Logger
}

func NewAzureBlobBackend(accountName, accountKey string, logger *zap.Logger) (*AzureBlobBackend, error) {
	// Create Azure Blob client
	// Return client
	return nil, nil
}

// Implement backend.StorageBackend interface
func (ab *AzureBlobBackend) CreateBucket(ctx context.Context, name string) error {
	return nil
}
// ... implement all interface methods
```

### 5. `internal/models/` - Data Models
**Files**:
- `internal/models/s3_responses.go` - S3 XML response structures
- `internal/models/azure_mapper.go` - Mapping utilities between S3 and Azure concepts

## Next Steps

1. Create each package directory and initial files
2. Implement handler functions for bucket and object operations
3. Implement Azure Blob backend using latest Azure SDK
4. Add error handling and response formatting
5. Integrate with main.go
6. Add comprehensive tests

## Key Integration Points

- `main.go` → creates config → initializes logger → creates backend → creates server → starts HTTP server
- HTTP handlers → verify SigV4 auth → call backend methods → format S3 responses
- Backend → translates S3 calls → Azure Blob SDK → returns results

