# Development Guide

This document provides implementation details and guidance for developing the S3→Azure proxy.

## Implementation Roadmap

### Phase 1: Core Infrastructure (CURRENT)
- [x] Project setup & Go module dependencies
- [x] Main entry point with graceful shutdown
- [ ] Config loader (env vars + file-based)
- [ ] Logging setup (zap integration)
- [ ] HTTP server with chi router

### Phase 2: S3 Frontend & Auth
- [ ] SigV4 authentication middleware
- [ ] Request canonicalization & validation
- [ ] S3 error response formatting (XML)
- [ ] Handler skeleton for bucket/object ops

### Phase 3: Backend Abstraction
- [ ] Storage backend interface definition
- [ ] Azure Blob implementation using latest SDK
- [ ] Container ↔ bucket mapping
- [ ] Blob name normalization

### Phase 4: Bucket Operations
- [ ] `PUT /{bucket}` – CreateBucket
- [ ] `DELETE /{bucket}` – DeleteBucket
- [ ] `GET /` – ListBuckets
- [ ] `GET /{bucket}?list-type=2` – ListObjectsV2

### Phase 5: Object Operations (Part 1)
- [ ] `PUT /{bucket}/{key}` – PutObject (single-part)
- [ ] `GET /{bucket}/{key}` – GetObject
- [ ] `HEAD /{bucket}/{key}` – HeadObject
- [ ] `DELETE /{bucket}/{key}` – DeleteObject

### Phase 6: Object Operations (Part 2) & Advanced
- [ ] Copy object (x-amz-copy-source)
- [ ] Multipart upload (initiate, upload part, complete, abort)
- [ ] CORS handling
- [ ] Performance optimization (streaming, timeouts)

### Phase 7: Testing & Deployment
- [ ] Unit tests
- [ ] Integration tests (with mock Azure backend)
- [ ] Load testing
- [ ] Docker image
- [ ] Documentation & examples

---

## Key Implementation Details

### 1. SigV4 Verification (using AWS SDK for Go v2)

**File**: `internal/auth/sigv4.go`

```go
type AuthVerifier struct {
    accessKey string
    secretKey string
    logger *zap.Logger
}

// VerifyRequest validates SigV4 signature on incoming request
func (av *AuthVerifier) VerifyRequest(r *http.Request) error {
    // 1. Extract Authorization header
    // 2. Parse access key from header
    // 3. Look up secret key (local mapping)
    // 4. Compute expected signature using aws/signer/v4
    // 5. Compare with provided signature
}
```

### 2. Storage Backend Interface

**File**: `internal/backend/backend.go`

```go
type StorageBackend interface {
    // Buckets
    CreateBucket(ctx context.Context, name string) error
    DeleteBucket(ctx context.Context, name string) error
    ListBuckets(ctx context.Context) ([]BucketInfo, error)
    
    // Objects
    PutObject(ctx, bucket, key string, r io.Reader, size int64, meta ObjectMeta) error
    GetObject(ctx, bucket, key string, rng *ByteRange) (io.ReadCloser, ObjectInfo, error)
    HeadObject(ctx, bucket, key string) (ObjectInfo, error)
    DeleteObject(ctx, bucket, key string) error
    
    // Listing
    ListObjectsV2(ctx, bucket string, opts ListOptions) (ListResult, error)
    
    // Multipart
    CreateMultipartUpload(ctx, bucket, key string, meta ObjectMeta) (uploadID string, err error)
    UploadPart(ctx, bucket, key, uploadID string, partNum int, r io.Reader, size int64) (etag string, err error)
    CompleteMultipartUpload(ctx, bucket, key, uploadID string, parts []CompletedPart) error
    AbortMultipartUpload(ctx, bucket, key, uploadID string) error
}
```

### 3. Azure Blob Implementation

**File**: `internal/backend/azureblob/client.go`

```go
type AzureBlobBackend struct {
    client     *azblob.Client
    accountKey string
    logger     *zap.Logger
}

func (ab *AzureBlobBackend) PutObject(ctx context.Context, bucket, key string, r io.Reader, size int64, meta ObjectMeta) error {
    // 1. Normalize key (S3 → Azure blob name)
    // 2. Map metadata to Azure headers
    // 3. Use UploadBuffer or UploadStream (depends on size)
    // 4. Handle errors → convert to S3 error codes
}
```

### 4. S3 Handler Structure

**File**: `internal/handler/s3.go`

```go
type S3Handler struct {
    backend StorageBackend
    logger  *zap.Logger
}

// Route handlers for bucket/object ops
func (s *S3Handler) PutObjectHandler(w http.ResponseWriter, r *http.Request) {
    bucket := chi.URLParam(r, "bucket")
    key := chi.URLParam(r, "key")
    
    // 1. Parse S3 headers (Content-Type, x-amz-meta-*, etc.)
    // 2. Verify SigV4 (via middleware)
    // 3. Call backend.PutObject()
    // 4. Write S3 response (ETag, etc.)
}
```

### 5. S3 XML Response Models

**File**: `internal/models/s3_responses.go`

Define Go structs that marshal to S3 XML:
```go
type ListBucketsResult struct {
    XMLName xml.Name   `xml:"ListAllMyBucketsResult"`
    Buckets []BucketInfo `xml:"Buckets>Bucket"`
    Owner   OwnerInfo    `xml:"Owner"`
}
```

---

## Azure Blob Storage Concepts → S3 Mapping

| S3 Concept | Azure Equivalent | Notes |
|---|---|---|
| Bucket | Container | 1:1 mapping |
| Object Key | Blob Name | Normalize path separators |
| Metadata | Metadata dict | Lowercase keys, limit values |
| ETag | ETag header | Usually MD5 or blob version ID |
| ACL | Access Policy | Limited; S3 ACLs → private only |
| Versioning | N/A (soft-deleted) | Not supported in v1 |
| Tagging | Metadata | Emulate via x-amz-meta-* |
| Server Encryption | CMK via MSI | Transparent to client |

---

## Error Mapping: S3 → Azure → S3

**File**: `internal/errors/mapper.go`

Common error mappings:
```
Azure NotFound                   → S3 NoSuchKey / NoSuchBucket
Azure ContainerAlreadyExists    → S3 BucketAlreadyExists / BucketAlreadyOwnedByYou
Azure Unauthorized              → S3 AccessDenied
Azure InvalidStorageUri         → S3 InvalidBucketName
Azure AuthorizationPermissionMismatch → S3 SignatureDoesNotMatch
```

---

## Performance Considerations

### Streaming
- Always stream uploads/downloads to avoid buffering full objects in memory
- Use `azblob.UploadStream` or `UploadBuffer` with sensible chunk sizes (1-4 MB)
- Forward `Range` headers to Azure for byte-range reads

### Connection Pooling
- Azure SDK manages connection pooling; tune via `ClientOptions.Transport`
- Configure max idle connections and timeouts

### Concurrency
- Multipart uploads in Go proxy can parallelize part uploads to Azure
- Consider thread pool or semaphore to limit concurrent parts

---

## Testing Strategy

### Unit Tests
- Mock `StorageBackend` interface for handler logic
- Test SigV4 verification in isolation
- Test S3 XML response marshalling

### Integration Tests
- Use Azure Storage Emulator (Azurite) or test account
- Test end-to-end requests via httptest
- Verify S3 client libraries work (boto3, aws-cli, etc.)

### Load Testing
- Use tools like `ab`, `wrk`, or `locust`
- Measure throughput, latency, error rates
- Profile memory/CPU usage

---

## References for Implementation

1. **AWS SDK for Go v2 SigV4**: https://pkg.go.dev/github.com/aws/aws-sdk-go-v2/aws/signer/v4
2. **Azure Blob Storage SDK**: https://github.com/Azure/azure-sdk-for-go/tree/main/sdk/storage/azblob
3. **S3 API Reference**: https://docs.aws.amazon.com/s3/latest/API/
4. **S3 Proxy (Java reference)**: https://github.com/gaul/s3proxy
5. **MinIO Azure Gateway** (deprecated, but useful reference): https://github.com/minio/minio/tree/master/cmd/gateway-azure

---

## Next Steps

1. Implement `internal/config` (environment variable loading)
2. Implement `internal/auth` (SigV4 verification)
3. Implement `internal/backend` interface
4. Implement `internal/backend/azureblob` (Azure client)
5. Implement handlers for bucket operations
6. Add tests at each step
7. Integrate with main.go and test end-to-end

