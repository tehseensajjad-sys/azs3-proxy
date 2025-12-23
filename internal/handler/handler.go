package handler

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/vibhansa-msft/s3-azure-proxy/internal/backend"
	"github.com/vibhansa-msft/s3-azure-proxy/internal/cache"
	"github.com/vibhansa-msft/s3-azure-proxy/internal/models"
	"github.com/vibhansa-msft/s3-azure-proxy/internal/telemetry"
)

// S3Handler handles all S3 API requests and converts them to backend storage operations.
// It acts as a bridge between S3 API semantics and the underlying storage backend.
// It includes optional local caching support for frequently accessed objects.
// It tracks comprehensive statistics for all S3 operations.
type S3Handler struct {
	backend      backend.StorageBackend   // Storage backend implementation (Azure Blob Storage)
	logger       *zap.Logger              // Logger for request and error logging
	cacheManager *cache.CacheManager      // Optional cache manager for storing/retrieving objects locally
	telMgr       *telemetry.Manager       // Optional telemetry manager for metrics export
	stats        *models.S3OperationStats // Statistics for S3 operations
}

// NewS3Handler creates and returns a new S3 API handler instance.
// It requires a storage backend implementation and logger for operation.
// Optional cache manager and telemetry manager can be set later.
func NewS3Handler(backend backend.StorageBackend, logger *zap.Logger) *S3Handler {
	return &S3Handler{
		backend:      backend,
		logger:       logger,
		cacheManager: nil,
		telMgr:       nil,
		stats:        &models.S3OperationStats{},
	}
}

// SetCacheManager sets the cache manager for the handler to enable local caching.
// This is called during server initialization if caching is configured.
func (h *S3Handler) SetCacheManager(cm *cache.CacheManager) {
	h.cacheManager = cm
}

// SetTelemetryManager sets the telemetry manager for the handler to enable metrics export.
// This is called during server initialization if telemetry is configured.
func (h *S3Handler) SetTelemetryManager(tm *telemetry.Manager) {
	h.telMgr = tm
}

// GetStats returns a snapshot of S3 operation statistics.
func (h *S3Handler) GetStats() models.StatsSnapshot {
	return h.stats.GetStats()
}

// generateCacheKey creates a cache key from bucket and object key.
// Format: "bucket/key" to uniquely identify each object across buckets.
func generateCacheKey(bucket, key string) string {
	return bucket + "/" + key
}

// writeErrorResponse writes an S3-formatted XML error response to the client.
// It sets appropriate HTTP status codes and formats the error message in S3 XML format.
func (h *S3Handler) writeErrorResponse(w http.ResponseWriter, err *models.S3Error) {
	w.Header().Set("Content-Type", "application/xml")
	w.WriteHeader(err.HTTPStatus())

	// Build S3-compatible error response
	resp := models.ErrorResponse{
		Code:      string(err.Code),
		Message:   err.Message,
		Resource:  err.Resource,
		RequestID: err.RequestID,
	}

	xmlData, _ := xml.Marshal(resp)
	_, _ = w.Write(xmlData)
}

// extractBucketAndKey extracts the bucket name and object key from the request URL.
// The key has the leading "/" stripped to match S3 semantics.
// Example: /bucket/folder/object.txt -> bucket: "bucket", key: "folder/object.txt"
func extractBucketAndKey(r *http.Request) (string, string) {
	bucket := chi.URLParam(r, "bucket")
	key := strings.TrimPrefix(chi.URLParam(r, "*"), "/")
	return bucket, key
}

// Bucket Operations

// ListBucketsHandler handles GET / (S3 ListBuckets operation).
// Returns all buckets owned by the account in S3 XML format.
func (h *S3Handler) ListBucketsHandler(w http.ResponseWriter, r *http.Request) {
	h.logger.Debug("ListBuckets request")

	// Fetch list of all buckets from backend storage
	buckets, err := h.backend.ListBuckets(r.Context())
	if err != nil {
		h.logger.Error("failed to list buckets", zap.Error(err))
		h.stats.RecordListBuckets(false)

		// Record telemetry
		if h.telMgr != nil {
			h.telMgr.RecordS3Request(r.Context(), "ListBuckets", false, "")
		}

		s3Err := &models.S3Error{
			Code:    models.InternalError,
			Message: "Failed to list buckets",
		}

		h.writeErrorResponse(w, s3Err)
		return
	}

	h.stats.RecordListBuckets(true)

	// Record telemetry
	if h.telMgr != nil {
		h.telMgr.RecordS3Request(r.Context(), "ListBuckets", true, "")
	}

	// Convert bucket names to S3 bucket list format
	bucketList := make([]models.Bucket, len(buckets))
	for i, name := range buckets {
		bucketList[i] = models.Bucket{
			Name:         name,
			CreationDate: time.Now().UTC().Format(time.RFC3339),
		}
	}

	// Build S3 ListBuckets response with owner information
	resp := models.ListBucketsResponse{
		Buckets: bucketList,
		Owner: models.Owner{
			ID:          "000000000000000000000000",
			DisplayName: "Anonymous",
		},
	}

	w.Header().Set("Content-Type", "application/xml")
	w.WriteHeader(http.StatusOK)
	xmlData, _ := xml.Marshal(resp)
	_, _ = w.Write(xmlData)
}

// HeadBucketHandler handles HEAD /{bucket} (S3 HeadBucket operation).
// Checks if a bucket exists and if the user has permission to access it.
// Returns 200 OK if bucket exists, 404 Not Found if it doesn't.
func (h *S3Handler) HeadBucketHandler(w http.ResponseWriter, r *http.Request) {
	bucket := chi.URLParam(r, "bucket")
	h.logger.Debug("HeadBucket request", zap.String("bucket", bucket))

	exists, err := h.backend.HeadBucket(r.Context(), bucket)
	if err != nil {
		h.logger.Error("failed to check bucket existence", zap.Error(err))
		// If error is not 404, return 500
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if !exists {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// CreateBucketHandler handles PUT /{bucket} (S3 CreateBucket operation).
// Creates a new bucket with the specified name.
func (h *S3Handler) CreateBucketHandler(w http.ResponseWriter, r *http.Request) {
	bucket := chi.URLParam(r, "bucket")
	h.logger.Debug("CreateBucket request", zap.String("bucket", bucket))

	// Create bucket in backend storage
	err := h.backend.CreateBucket(r.Context(), bucket)
	if err != nil {
		h.logger.Error("failed to create bucket", zap.Error(err), zap.String("bucket", bucket))
		// Record telemetry
		if h.telMgr != nil {
			h.telMgr.RecordS3Request(r.Context(), "CreateBucket", false, "")
		}
		errMsg := err.Error()
		s3ErrCode := models.AzureErrorToS3(errMsg)
		s3Err := &models.S3Error{
			Code:     s3ErrCode,
			Message:  errMsg,
			Resource: "/" + bucket,
		}

		h.writeErrorResponse(w, s3Err)
		return
	}

	w.WriteHeader(http.StatusOK)
	// Record telemetry
	if h.telMgr != nil {
		h.telMgr.RecordS3Request(r.Context(), "CreateBucket", true, "")
	}
}

// DeleteBucketHandler handles DELETE /{bucket} (S3 DeleteBucket operation).
// Deletes a bucket (must be empty, no objects).
func (h *S3Handler) DeleteBucketHandler(w http.ResponseWriter, r *http.Request) {
	bucket := chi.URLParam(r, "bucket")
	h.logger.Debug("DeleteBucket request", zap.String("bucket", bucket))

	// Delete bucket from backend storage
	err := h.backend.DeleteBucket(r.Context(), bucket)
	if err != nil {
		h.logger.Error("failed to delete bucket", zap.Error(err), zap.String("bucket", bucket))

		errMsg := err.Error()
		s3ErrCode := models.AzureErrorToS3(errMsg)
		s3Err := &models.S3Error{
			Code:     s3ErrCode,
			Message:  errMsg,
			Resource: "/" + bucket,
		}

		h.writeErrorResponse(w, s3Err)
		return
	}

	h.stats.RecordDeleteBucket(true)
	w.WriteHeader(http.StatusNoContent)
	if h.telMgr != nil {
		h.telMgr.RecordS3Request(r.Context(), "DeleteBucket", true, "")
	}
}

// ListObjectsV2Handler handles GET /{bucket}/?list-type=2
// Lists objects in the bucket with optional prefix filter.
func (h *S3Handler) ListObjectsV2Handler(w http.ResponseWriter, r *http.Request) {
	bucket := chi.URLParam(r, "bucket")
	prefix := r.URL.Query().Get("prefix")
	h.logger.Debug("ListObjectsV2 request", zap.String("bucket", bucket), zap.String("prefix", prefix))

	// Fetch objects from backend with optional prefix filtering
	objects, err := h.backend.ListObjects(r.Context(), bucket, prefix)
	if err != nil {
		h.logger.Error("failed to list objects", zap.Error(err), zap.String("bucket", bucket))
		h.stats.RecordListObjectsV2(false)
		if h.telMgr != nil {
			h.telMgr.RecordS3Request(r.Context(), "ListObjectsV2", false, "")
		}

		errMsg := err.Error()
		s3ErrCode := models.AzureErrorToS3(errMsg)
		s3Err := &models.S3Error{
			Code:     s3ErrCode,
			Message:  errMsg,
			Resource: "/" + bucket,
		}

		h.writeErrorResponse(w, s3Err)
		return
	}

	// Convert object names to S3 object list format
	objList := make([]models.Object, len(objects))
	for i, key := range objects {
		objList[i] = models.Object{
			Key:          key,
			LastModified: time.Now().UTC().Format(time.RFC3339),
			ETag:         "\"0\"",
			Size:         0,
			StorageClass: "STANDARD",
		}
	}

	// Build and return S3 ListObjects response
	resp := models.ListObjectsResponse{
		Name:        bucket,
		Prefix:      prefix,
		MaxKeys:     1000,
		IsTruncated: false,
		Contents:    objList,
	}

	w.Header().Set("Content-Type", "application/xml")
	w.WriteHeader(http.StatusOK)
	xmlData, _ := xml.Marshal(resp)
	_, _ = w.Write(xmlData)
	h.stats.RecordListObjectsV2(true)
	if h.telMgr != nil {
		h.telMgr.RecordS3Request(r.Context(), "ListObjectsV2", true, "")
	}
}

// Object Operations

// PutObjectHandler handles PUT /{bucket}/{key} (S3 PutObject operation).
// Uploads a complete object to the bucket. Supports single-part uploads.
func (h *S3Handler) PutObjectHandler(w http.ResponseWriter, r *http.Request) {
	bucket, key := extractBucketAndKey(r)
	contentLength := r.ContentLength
	copySource := r.Header.Get("x-amz-copy-source")

	h.logger.Debug("PutObject request",
		zap.String("bucket", bucket),
		zap.String("key", key),
		zap.Int64("content_length", contentLength),
		zap.String("copy_source", copySource))

	// Ensure this is an object operation, not a bucket operation
	if key == "" {
		h.CreateBucketHandler(w, r)
		return
	}

	var err error
	if copySource != "" {
		// Handle CopyObject
		decodedSource, decodeErr := url.QueryUnescape(copySource)
		if decodeErr != nil {
			h.logger.Error("failed to decode copy source", zap.Error(decodeErr))
			h.writeErrorResponse(w, &models.S3Error{
				Code:    models.InvalidArgument,
				Message: "Invalid copy source encoding",
			})
			return
		}

		// Format: /bucket/key or bucket/key
		decodedSource = strings.TrimPrefix(decodedSource, "/")
		parts := strings.SplitN(decodedSource, "/", 2)
		if len(parts) != 2 {
			h.writeErrorResponse(w, &models.S3Error{
				Code:    models.InvalidArgument,
				Message: "Invalid copy source format",
			})
			return
		}
		srcBucket, srcKey := parts[0], parts[1]

		err = h.backend.CopyObject(r.Context(), srcBucket, srcKey, bucket, key)
	} else {
		err = h.backend.PutObject(r.Context(), bucket, key, r.Body)
	}

	if err != nil {
		h.logger.Error("failed to put object",
			zap.Error(err),
			zap.String("bucket", bucket),
			zap.String("key", key),
			zap.Int64("content_length", contentLength))
		h.stats.RecordPutObject(false)

		// Record telemetry
		if h.telMgr != nil {
			h.telMgr.RecordS3Request(r.Context(), "PutObject", false, "")
		}

		errMsg := err.Error()
		s3ErrCode := models.AzureErrorToS3(errMsg)
		s3Err := &models.S3Error{
			Code:     s3ErrCode,
			Message:  errMsg,
			Resource: "/" + bucket + "/" + key,
		}
		h.writeErrorResponse(w, s3Err)
		return
	}

	// If caching is enabled, invalidate the old cached version of this object
	// so that the next GET request will fetch the fresh version from backend
	if h.cacheManager != nil {
		cacheKey := generateCacheKey(bucket, key)
		if err := h.cacheManager.InvalidateObject(cacheKey); err != nil {
			h.logger.Warn("failed to invalidate cache after put",
				zap.Error(err),
				zap.String("cache_key", cacheKey))
		}
	}

	h.logger.Info("object uploaded successfully",
		zap.String("bucket", bucket),
		zap.String("key", key),
		zap.Int64("content_length", contentLength))

	h.stats.RecordPutObject(true)

	// Record telemetry
	if h.telMgr != nil {
		h.telMgr.RecordS3Request(r.Context(), "PutObject", true, "")
	}

	if copySource != "" {
		resp := models.CopyObjectResult{
			LastModified: time.Now().UTC().Format(time.RFC3339),
			ETag:         "\"0\"", // Placeholder ETag
		}
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusOK)
		xmlData, _ := xml.Marshal(resp)
		_, _ = w.Write(xmlData)
	} else {
		w.Header().Set("ETag", "\"0\"")
		w.WriteHeader(http.StatusOK)
	}
}

// GetObjectHandler handles GET /{bucket}/{key}
func (h *S3Handler) GetObjectHandler(w http.ResponseWriter, r *http.Request) {
	bucket, key := extractBucketAndKey(r)
	h.logger.Debug("GetObject request",
		zap.String("bucket", bucket),
		zap.String("key", key))

	if key == "" {
		// This is a list operation, not a get object operation
		h.ListObjectsV2Handler(w, r)
		return
	}

	// If caching is enabled, check cache first
	if h.cacheManager != nil {
		cacheKey := generateCacheKey(bucket, key)
		cachedFilePath, err := h.cacheManager.GetObjectFromCache(cacheKey)
		if err == nil && cachedFilePath != "" {
			// Found in cache! Serve from local file
			h.logger.Debug("Serving object from cache",
				zap.String("bucket", bucket),
				zap.String("key", key),
				zap.String("cache_path", cachedFilePath))

			// Read and serve cached file
			data, err := os.ReadFile(cachedFilePath)
			if err == nil {
				w.Header().Set("Content-Type", "application/octet-stream")
				w.Header().Set("ETag", "\"0\"")
				w.Header().Set("X-Cache-Hit", "true")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(data)
				return
			}
			// If we can't read the cached file, fall through to fetch from backend
			h.logger.Warn("failed to read cached file, fetching from backend",
				zap.Error(err),
				zap.String("cache_path", cachedFilePath))
		}
	}

	// Not in cache or caching disabled - fetch from backend
	body, err := h.backend.GetObject(r.Context(), bucket, key)
	if err != nil {
		h.logger.Error("failed to get object",
			zap.Error(err),
			zap.String("bucket", bucket),
			zap.String("key", key))
		h.stats.RecordGetObject(false)
		if h.telMgr != nil {
			h.telMgr.RecordS3Request(r.Context(), "GetObject", false, "")
		}
		errMsg := err.Error()
		s3ErrCode := models.AzureErrorToS3(errMsg)
		s3Err := &models.S3Error{
			Code:     s3ErrCode,
			Message:  errMsg,
			Resource: "/" + bucket + "/" + key,
		}
		h.writeErrorResponse(w, s3Err)
		return
	}
	defer func() { _ = body.Close() }()

	h.logger.Info("object downloaded successfully",
		zap.String("bucket", bucket),
		zap.String("key", key))

	// If caching is enabled, cache the object for future requests
	// We buffer the response to avoid race conditions with async caching
	var objectData []byte
	if h.cacheManager != nil {
		var readErr error
		// Read entire object into memory to cache it
		objectData, readErr = io.ReadAll(body)
		if readErr != nil {
			h.logger.Error("failed to read object for caching",
				zap.Error(readErr),
				zap.String("bucket", bucket),
				zap.String("key", key))
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Header().Set("ETag", "\"0\"")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// Cache the object asynchronously to avoid blocking response
		cacheKey := generateCacheKey(bucket, key)
		go func() {
			if err := h.cacheManager.CacheObject(cacheKey, objectData); err != nil {
				h.logger.Warn("failed to cache object after get",
					zap.Error(err),
					zap.String("cache_key", cacheKey))
			}
		}()
	} else {
		// If caching disabled, read the object normally
		var readErr error
		objectData, readErr = io.ReadAll(body)
		if readErr != nil {
			h.logger.Error("failed to read object",
				zap.Error(readErr),
				zap.String("bucket", bucket),
				zap.String("key", key))
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Header().Set("ETag", "\"0\"")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("ETag", "\"0\"")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(objectData)
	h.stats.RecordGetObject(true)
	if h.telMgr != nil {
		h.telMgr.RecordS3Request(r.Context(), "GetObject", true, "")
	}
}

// HeadObjectHandler handles HEAD /{bucket}/{key}
func (h *S3Handler) HeadObjectHandler(w http.ResponseWriter, r *http.Request) {
	bucket, key := extractBucketAndKey(r)
	h.logger.Debug("HeadObject request", zap.String("bucket", bucket), zap.String("key", key))

	// If caching is enabled, check cache first
	if h.cacheManager != nil {
		cacheKey := generateCacheKey(bucket, key)
		cachedFilePath, err := h.cacheManager.GetObjectFromCache(cacheKey)
		if err == nil && cachedFilePath != "" {
			// Found in cache! Return metadata without serving content
			if info, err := os.Stat(cachedFilePath); err == nil {
				h.logger.Debug("HeadObject served from cache",
					zap.String("bucket", bucket),
					zap.String("key", key),
					zap.Int64("size", info.Size()))

				w.Header().Set("Content-Type", "application/octet-stream")
				w.Header().Set("Content-Length", fmt.Sprintf("%d", info.Size()))
				w.Header().Set("ETag", "\"0\"")
				w.Header().Set("X-Cache-Hit", "true")
				w.WriteHeader(http.StatusOK)
				return
			}
		}
	}

	exists, err := h.backend.HeadObject(r.Context(), bucket, key)
	if err != nil {
		h.logger.Error("failed to head object", zap.Error(err), zap.String("bucket", bucket), zap.String("key", key))
		h.stats.RecordHeadObject(false)
		if h.telMgr != nil {
			h.telMgr.RecordS3Request(r.Context(), "HeadObject", false, "")
		}
		errMsg := err.Error()
		s3ErrCode := models.AzureErrorToS3(errMsg)
		s3Err := &models.S3Error{
			Code:     s3ErrCode,
			Message:  errMsg,
			Resource: "/" + bucket + "/" + key,
		}
		h.writeErrorResponse(w, s3Err)
		return
	}

	if !exists {
		h.stats.RecordHeadObject(false)
		if h.telMgr != nil {
			h.telMgr.RecordS3Request(r.Context(), "HeadObject", false, "NoSuchKey")
		}
		s3Err := &models.S3Error{
			Code:     models.NoSuchKey,
			Message:  "The specified key does not exist.",
			Resource: "/" + bucket + "/" + key,
		}
		h.writeErrorResponse(w, s3Err)
		return
	}

	w.Header().Set("ETag", "\"0\"")
	w.WriteHeader(http.StatusOK)
	h.stats.RecordHeadObject(true)
	if h.telMgr != nil {
		h.telMgr.RecordS3Request(r.Context(), "HeadObject", true, "")
	}
}

// DeleteObjectHandler handles DELETE /{bucket}/{key}
func (h *S3Handler) DeleteObjectHandler(w http.ResponseWriter, r *http.Request) {
	bucket, key := extractBucketAndKey(r)
	h.logger.Debug("DeleteObject request", zap.String("bucket", bucket), zap.String("key", key))

	if key == "" {
		// This is a bucket operation, not an object operation
		h.DeleteBucketHandler(w, r)
		return
	}

	err := h.backend.DeleteObject(r.Context(), bucket, key)
	if err != nil {
		h.logger.Error("failed to delete object", zap.Error(err), zap.String("bucket", bucket), zap.String("key", key))
		h.stats.RecordDeleteObject(false)
		if h.telMgr != nil {
			h.telMgr.RecordS3Request(r.Context(), "DeleteObject", false, "")
		}
		errMsg := err.Error()
		s3ErrCode := models.AzureErrorToS3(errMsg)
		s3Err := &models.S3Error{
			Code:     s3ErrCode,
			Message:  errMsg,
			Resource: "/" + bucket + "/" + key,
		}
		h.writeErrorResponse(w, s3Err)
		return
	}

	// If caching is enabled, invalidate the deleted object from cache
	if h.cacheManager != nil {
		cacheKey := generateCacheKey(bucket, key)
		if err := h.cacheManager.InvalidateObject(cacheKey); err != nil {
			h.logger.Warn("failed to invalidate cache after delete",
				zap.Error(err),
				zap.String("cache_key", cacheKey))
		}
	}

	h.logger.Info("object deleted successfully", zap.String("bucket", bucket), zap.String("key", key))
	h.stats.RecordDeleteObject(true)
	w.WriteHeader(http.StatusNoContent)
	if h.telMgr != nil {
		h.telMgr.RecordS3Request(r.Context(), "DeleteObject", true, "")
	}
}

// PostObjectHandler handles POST /{bucket}/{key} (multipart upload)
// - POST /{bucket}/{key}?uploads - Initiate multipart upload
// - POST /{bucket}/{key}?uploadId=... - Complete multipart upload
func (h *S3Handler) PostObjectHandler(w http.ResponseWriter, r *http.Request) {
	bucket, key := extractBucketAndKey(r)
	h.logger.Debug("PostObject request", zap.String("bucket", bucket), zap.String("key", key))

	// Check if this is an upload completion request
	uploadID := r.URL.Query().Get("uploadId")
	if uploadID != "" {
		h.CompleteMultipartUploadHandler(w, r)
		return
	}

	// Check if this is an initiate request
	if r.URL.Query().Get("uploads") == "" && uploadID == "" {
		// Check for list multipart uploads
		if key == "" {
			h.ListMultipartUploadsHandler(w, r)
			return
		}
	}

	h.InitiateMultipartUploadHandler(w, r)
}

// InitiateMultipartUploadHandler handles POST /{bucket}/{key}?uploads
func (h *S3Handler) InitiateMultipartUploadHandler(w http.ResponseWriter, r *http.Request) {
	bucket, key := extractBucketAndKey(r)
	h.logger.Debug("InitiateMultipartUpload request", zap.String("bucket", bucket), zap.String("key", key))

	if key == "" {
		s3Err := &models.S3Error{
			Code:     models.InternalError,
			Message:  "Key is required for multipart upload",
			Resource: "/" + bucket,
		}
		h.writeErrorResponse(w, s3Err)
		return
	}

	uploadID, err := h.backend.InitiateMultipartUpload(r.Context(), bucket, key)
	if err != nil {
		h.logger.Error("failed to initiate multipart upload", zap.Error(err))
		h.stats.RecordInitiateMultipart(false)
		if h.telMgr != nil {
			h.telMgr.RecordS3Request(r.Context(), "InitiateMultipartUpload", false, "")
		}
		s3Err := &models.S3Error{
			Code:    models.InternalError,
			Message: err.Error(),
		}
		h.writeErrorResponse(w, s3Err)
		return
	}

	w.Header().Set("Content-Type", "application/xml")
	w.WriteHeader(http.StatusOK)
	resp := models.InitiateMultipartUploadResponse{
		Bucket:   bucket,
		Key:      key,
		UploadID: uploadID,
	}
	xmlData, _ := xml.Marshal(resp)
	_, _ = w.Write(xmlData)
	h.stats.RecordInitiateMultipart(true)
	if h.telMgr != nil {
		h.telMgr.RecordS3Request(r.Context(), "InitiateMultipartUpload", true, "")
	}
}

// UploadPartHandler handles PUT /{bucket}/{key}?partNumber=X&uploadId=...
func (h *S3Handler) UploadPartHandler(w http.ResponseWriter, r *http.Request) {
	bucket, key := extractBucketAndKey(r)
	uploadID := r.URL.Query().Get("uploadId")
	partNumberStr := r.URL.Query().Get("partNumber")

	h.logger.Debug("UploadPart request",
		zap.String("bucket", bucket),
		zap.String("key", key),
		zap.String("uploadId", uploadID),
		zap.String("partNumber", partNumberStr))

	if uploadID == "" {
		s3Err := &models.S3Error{
			Code:    models.InternalError,
			Message: "uploadId is required",
		}
		h.writeErrorResponse(w, s3Err)
		return
	}

	if partNumberStr == "" {
		s3Err := &models.S3Error{
			Code:    models.InternalError,
			Message: "partNumber is required",
		}
		h.writeErrorResponse(w, s3Err)
		return
	}

	var partNumber int
	partNum, err := strconv.Atoi(partNumberStr)
	if err != nil || partNum < 1 || partNum > 10000 {
		s3Err := &models.S3Error{
			Code:    models.InternalError,
			Message: "partNumber must be an integer between 1 and 10000",
		}
		h.writeErrorResponse(w, s3Err)
		return
	}
	partNumber = partNum

	etag, err := h.backend.UploadPart(r.Context(), bucket, key, uploadID, partNumber, r.Body)
	if err != nil {
		h.logger.Error("failed to upload part", zap.Error(err))
		h.stats.RecordUploadPart(false)
		if h.telMgr != nil {
			h.telMgr.RecordS3Request(r.Context(), "UploadPart", false, "")
		}
		s3Err := &models.S3Error{
			Code:    models.InternalError,
			Message: err.Error(),
		}
		h.writeErrorResponse(w, s3Err)
		return
	}

	w.Header().Set("ETag", etag)
	w.WriteHeader(http.StatusOK)
	h.stats.RecordUploadPart(true)
	if h.telMgr != nil {
		h.telMgr.RecordS3Request(r.Context(), "UploadPart", true, "")
	}
}

// CompleteMultipartUploadHandler handles POST /{bucket}/{key}?uploadId=...
func (h *S3Handler) CompleteMultipartUploadHandler(w http.ResponseWriter, r *http.Request) {
	bucket, key := extractBucketAndKey(r)
	uploadID := r.URL.Query().Get("uploadId")

	h.logger.Debug("CompleteMultipartUpload request",
		zap.String("bucket", bucket),
		zap.String("key", key),
		zap.String("uploadId", uploadID))

	if uploadID == "" {
		s3Err := &models.S3Error{
			Code:    models.InternalError,
			Message: "uploadId is required",
		}
		h.writeErrorResponse(w, s3Err)
		return
	}

	// Parse the request body to get part ETags
	var completeReq struct {
		Parts []struct {
			PartNumber int    `xml:"PartNumber"`
			ETag       string `xml:"ETag"`
		} `xml:"Part"`
	}

	if err := xml.NewDecoder(r.Body).Decode(&completeReq); err != nil {
		h.logger.Error("failed to parse complete multipart upload request", zap.Error(err))
		s3Err := &models.S3Error{
			Code:    models.InternalError,
			Message: "Invalid request body",
		}
		h.writeErrorResponse(w, s3Err)
		return
	}

	// Build part ETags map
	partETags := make(map[int]string)
	for _, part := range completeReq.Parts {
		partETags[part.PartNumber] = part.ETag
	}

	etag, err := h.backend.CompleteMultipartUpload(r.Context(), bucket, key, uploadID, partETags)
	if err != nil {
		h.logger.Error("failed to complete multipart upload", zap.Error(err))
		h.stats.RecordCompleteMultipart(false)
		if h.telMgr != nil {
			h.telMgr.RecordS3Request(r.Context(), "CompleteMultipartUpload", false, "")
		}
		s3Err := &models.S3Error{
			Code:    models.InternalError,
			Message: err.Error(),
		}
		h.writeErrorResponse(w, s3Err)
		return
	}

	w.Header().Set("Content-Type", "application/xml")
	w.WriteHeader(http.StatusOK)
	resp := models.CompleteMultipartUploadResponse{
		Bucket: bucket,
		Key:    key,
		ETag:   etag,
	}
	xmlData, _ := xml.Marshal(resp)
	_, _ = w.Write(xmlData)
	h.stats.RecordCompleteMultipart(true)
	if h.telMgr != nil {
		h.telMgr.RecordS3Request(r.Context(), "CompleteMultipartUpload", true, "")
	}
}

// AbortMultipartUploadHandler handles DELETE /{bucket}/{key}?uploadId=...
func (h *S3Handler) AbortMultipartUploadHandler(w http.ResponseWriter, r *http.Request) {
	bucket, key := extractBucketAndKey(r)
	uploadID := r.URL.Query().Get("uploadId")

	h.logger.Debug("AbortMultipartUpload request",
		zap.String("bucket", bucket),
		zap.String("key", key),
		zap.String("uploadId", uploadID))

	if uploadID == "" {
		s3Err := &models.S3Error{
			Code:    models.InternalError,
			Message: "uploadId is required",
		}
		h.writeErrorResponse(w, s3Err)
		return
	}

	err := h.backend.AbortMultipartUpload(r.Context(), bucket, key, uploadID)
	if err != nil {
		h.logger.Error("failed to abort multipart upload", zap.Error(err))
		h.stats.RecordAbortMultipart(false)
		if h.telMgr != nil {
			h.telMgr.RecordS3Request(r.Context(), "AbortMultipartUpload", false, "")
		}
		s3Err := &models.S3Error{
			Code:    models.InternalError,
			Message: err.Error(),
		}
		h.writeErrorResponse(w, s3Err)
		return
	}

	h.stats.RecordAbortMultipart(true)
	if h.telMgr != nil {
		h.telMgr.RecordS3Request(r.Context(), "AbortMultipartUpload", true, "")
	}
	w.WriteHeader(http.StatusNoContent)
}

// ListPartsHandler handles GET /{bucket}/{key}?uploadId=...
func (h *S3Handler) ListPartsHandler(w http.ResponseWriter, r *http.Request) {
	bucket, key := extractBucketAndKey(r)
	uploadID := r.URL.Query().Get("uploadId")

	h.logger.Debug("ListParts request",
		zap.String("bucket", bucket),
		zap.String("key", key),
		zap.String("uploadId", uploadID))

	if uploadID == "" {
		s3Err := &models.S3Error{
			Code:    models.InternalError,
			Message: "uploadId is required",
		}
		h.writeErrorResponse(w, s3Err)
		return
	}

	parts, err := h.backend.ListParts(r.Context(), bucket, key, uploadID)
	if err != nil {
		h.logger.Error("failed to list parts", zap.Error(err))
		h.stats.RecordListParts(false)
		if h.telMgr != nil {
			h.telMgr.RecordS3Request(r.Context(), "ListParts", false, "")
		}
		s3Err := &models.S3Error{
			Code:    models.InternalError,
			Message: err.Error(),
		}
		h.writeErrorResponse(w, s3Err)
		return
	}

	w.Header().Set("Content-Type", "application/xml")
	w.WriteHeader(http.StatusOK)

	partList := make([]models.Part, 0, len(parts))
	for _, p := range parts {
		partMap := p.(map[string]interface{})
		partList = append(partList, models.Part{
			PartNumber: partMap["PartNumber"].(int),
			ETag:       partMap["ETag"].(string),
			Size:       partMap["Size"].(int64),
		})
	}

	resp := models.ListPartsResponse{
		Bucket:       bucket,
		Key:          key,
		UploadID:     uploadID,
		StorageClass: "STANDARD",
		MaxParts:     1000,
		IsTruncated:  false,
		Parts:        partList,
	}
	xmlData, _ := xml.Marshal(resp)
	_, _ = w.Write(xmlData)
	h.stats.RecordListParts(true)
	if h.telMgr != nil {
		h.telMgr.RecordS3Request(r.Context(), "ListParts", true, "")
	}
}

// ListMultipartUploadsHandler handles GET /{bucket}?uploads
func (h *S3Handler) ListMultipartUploadsHandler(w http.ResponseWriter, r *http.Request) {
	bucket := chi.URLParam(r, "bucket")
	h.logger.Debug("ListMultipartUploads request", zap.String("bucket", bucket))

	uploads, err := h.backend.ListMultipartUploads(r.Context(), bucket)
	if err != nil {
		h.logger.Error("failed to list multipart uploads", zap.Error(err))
		h.stats.RecordListMultipartUploads(false)
		if h.telMgr != nil {
			h.telMgr.RecordS3Request(r.Context(), "ListMultipartUploads", false, "")
		}
		s3Err := &models.S3Error{
			Code:    models.InternalError,
			Message: err.Error(),
		}
		h.writeErrorResponse(w, s3Err)
		return
	}

	w.Header().Set("Content-Type", "application/xml")
	w.WriteHeader(http.StatusOK)

	uploadList := make([]models.UploadInfo, 0, len(uploads))
	for _, u := range uploads {
		uploadMap := u.(map[string]interface{})
		uploadList = append(uploadList, models.UploadInfo{
			Key:          uploadMap["Key"].(string),
			UploadID:     uploadMap["UploadID"].(string),
			Initiated:    uploadMap["Initiated"].(string),
			StorageClass: "STANDARD",
		})
	}

	h.stats.RecordListMultipartUploads(true)
	resp := models.ListMultipartUploadsResponse{
		Bucket:      bucket,
		MaxUploads:  1000,
		IsTruncated: false,
		Uploads:     uploadList,
	}
	xmlData, _ := xml.Marshal(resp)
	_, _ = w.Write(xmlData)
	if h.telMgr != nil {
		h.telMgr.RecordS3Request(r.Context(), "ListMultipartUploads", true, "")
	}
}

// EnableVersioningHandler enables versioning on a bucket
func (h *S3Handler) EnableVersioningHandler(w http.ResponseWriter, r *http.Request) {
	bucket := chi.URLParam(r, "bucket")
	h.logger.Debug("EnableVersioning request", zap.String("bucket", bucket))

	err := h.backend.EnableVersioning(r.Context(), bucket)
	if err != nil {
		h.logger.Error("failed to enable versioning", zap.String("bucket", bucket), zap.Error(err))
		h.stats.RecordEnableVersioning(false)
		if h.telMgr != nil {
			h.telMgr.RecordS3Request(r.Context(), "EnableVersioning", false, "")
		}
		errMsg := err.Error()
		s3ErrCode := models.AzureErrorToS3(errMsg)
		s3Err := &models.S3Error{
			Code:     s3ErrCode,
			Message:  errMsg,
			Resource: "/" + bucket,
		}
		h.writeErrorResponse(w, s3Err)
		return
	}

	w.Header().Set("Content-Type", "application/xml")
	w.WriteHeader(http.StatusOK)
	resp := models.VersioningConfiguration{
		Status: "Enabled",
	}
	xmlData, _ := xml.Marshal(resp)
	_, _ = w.Write(xmlData)
	h.stats.RecordEnableVersioning(true)
	if h.telMgr != nil {
		h.telMgr.RecordS3Request(r.Context(), "EnableVersioning", true, "")
	}
}

// GetVersioningHandler gets the versioning status of a bucket
func (h *S3Handler) GetVersioningHandler(w http.ResponseWriter, r *http.Request) {
	bucket := chi.URLParam(r, "bucket")
	h.logger.Debug("GetVersioning request", zap.String("bucket", bucket))

	enabled, err := h.backend.GetVersioning(r.Context(), bucket)
	if err != nil {
		h.logger.Error("failed to get versioning", zap.String("bucket", bucket), zap.Error(err))
		h.stats.RecordGetVersioning(false)
		if h.telMgr != nil {
			h.telMgr.RecordS3Request(r.Context(), "GetVersioning", false, "")
		}
		errMsg := err.Error()
		s3ErrCode := models.AzureErrorToS3(errMsg)
		s3Err := &models.S3Error{
			Code:     s3ErrCode,
			Message:  errMsg,
			Resource: "/" + bucket,
		}
		h.writeErrorResponse(w, s3Err)
		return
	}

	w.Header().Set("Content-Type", "application/xml")
	w.WriteHeader(http.StatusOK)

	status := ""
	if enabled {
		status = "Enabled"
	} else {
		status = "Suspended"
	}

	resp := models.VersioningConfiguration{
		Status: status,
	}
	xmlData, _ := xml.Marshal(resp)
	_, _ = w.Write(xmlData)
	h.stats.RecordGetVersioning(true)
	if h.telMgr != nil {
		h.telMgr.RecordS3Request(r.Context(), "GetVersioning", true, "")
	}
}

// ListObjectVersionsHandler lists all versions of objects in a bucket
func (h *S3Handler) ListObjectVersionsHandler(w http.ResponseWriter, r *http.Request) {
	bucket := chi.URLParam(r, "bucket")
	prefix := r.URL.Query().Get("prefix")
	h.logger.Debug("ListObjectVersions request", zap.String("bucket", bucket), zap.String("prefix", prefix))

	versions, err := h.backend.ListObjectVersions(r.Context(), bucket, prefix)
	if err != nil {
		h.logger.Error("failed to list object versions", zap.String("bucket", bucket), zap.Error(err))
		h.stats.RecordListObjectVersions(false)
		if h.telMgr != nil {
			h.telMgr.RecordS3Request(r.Context(), "ListObjectVersions", false, "")
		}
		errMsg := err.Error()
		s3ErrCode := models.AzureErrorToS3(errMsg)
		s3Err := &models.S3Error{
			Code:     s3ErrCode,
			Message:  errMsg,
			Resource: "/" + bucket,
		}
		h.writeErrorResponse(w, s3Err)
		return
	}

	w.Header().Set("Content-Type", "application/xml")
	w.WriteHeader(http.StatusOK)

	var versionList []models.ObjectVersionXML
	for _, v := range versions {
		objVersion := v.(backend.ObjectVersion)
		versionList = append(versionList, models.ObjectVersionXML{
			Key:          objVersion.Key,
			VersionID:    objVersion.VersionID,
			IsLatest:     objVersion.IsLatest,
			LastModified: objVersion.Modified,
			ETag:         objVersion.ETag,
			Size:         objVersion.Size,
			StorageClass: "STANDARD",
		})
	}

	resp := models.ListObjectVersionsResponse{
		Name:        bucket,
		Prefix:      prefix,
		IsTruncated: false,
		Versions:    versionList,
	}
	xmlData, _ := xml.Marshal(resp)
	_, _ = w.Write(xmlData)
	h.stats.RecordListObjectVersions(true)
	if h.telMgr != nil {
		h.telMgr.RecordS3Request(r.Context(), "ListObjectVersions", true, "")
	}
}

// GetObjectVersionHandler retrieves a specific version of an object
func (h *S3Handler) GetObjectVersionHandler(w http.ResponseWriter, r *http.Request) {
	bucket := chi.URLParam(r, "bucket")
	key := strings.TrimPrefix(chi.URLParam(r, "*"), "/")
	versionID := r.URL.Query().Get("versionId")
	h.logger.Debug("GetObjectVersion request", zap.String("bucket", bucket), zap.String("key", key), zap.String("versionId", versionID))

	reader, err := h.backend.GetObjectVersion(r.Context(), bucket, key, versionID)
	if err != nil {
		h.logger.Error("failed to get object version", zap.String("bucket", bucket), zap.String("key", key), zap.Error(err))
		h.stats.RecordGetObjectVersion(false)
		if h.telMgr != nil {
			h.telMgr.RecordS3Request(r.Context(), "GetObjectVersion", false, "")
		}
		errMsg := err.Error()
		s3ErrCode := models.AzureErrorToS3(errMsg)
		s3Err := &models.S3Error{
			Code:     s3ErrCode,
			Message:  errMsg,
			Resource: "/" + bucket + "/" + key,
		}
		h.writeErrorResponse(w, s3Err)
		return
	}
	defer func() { _ = reader.Close() }()

	w.Header().Set("Content-Type", "application/octet-stream")
	if versionID != "" {
		w.Header().Set("x-amz-version-id", versionID)
	}
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, reader)
	h.stats.RecordGetObjectVersion(true)
	if h.telMgr != nil {
		h.telMgr.RecordS3Request(r.Context(), "GetObjectVersion", true, "")
	}
}

// DeleteObjectVersionHandler deletes a specific version of an object
func (h *S3Handler) DeleteObjectVersionHandler(w http.ResponseWriter, r *http.Request) {
	bucket := chi.URLParam(r, "bucket")
	key := strings.TrimPrefix(chi.URLParam(r, "*"), "/")
	versionID := r.URL.Query().Get("versionId")
	h.logger.Debug("DeleteObjectVersion request", zap.String("bucket", bucket), zap.String("key", key), zap.String("versionId", versionID))

	err := h.backend.DeleteObjectVersion(r.Context(), bucket, key, versionID)
	if err != nil {
		h.logger.Error("failed to delete object version", zap.String("bucket", bucket), zap.String("key", key), zap.Error(err))
		h.stats.RecordDeleteObjectVersion(false)
		if h.telMgr != nil {
			h.telMgr.RecordS3Request(r.Context(), "DeleteObjectVersion", false, "")
		}
		errMsg := err.Error()
		s3ErrCode := models.AzureErrorToS3(errMsg)
		s3Err := &models.S3Error{
			Code:     s3ErrCode,
			Message:  errMsg,
			Resource: "/" + bucket + "/" + key,
		}
		h.writeErrorResponse(w, s3Err)
		return
	}

	h.stats.RecordDeleteObjectVersion(true)
	if h.telMgr != nil {
		h.telMgr.RecordS3Request(r.Context(), "DeleteObjectVersion", true, "")
	}
	w.WriteHeader(http.StatusNoContent)
}

// DeleteObjectsHandler handles POST /{bucket}?delete
// Deletes multiple objects in a single request.
func (h *S3Handler) DeleteObjectsHandler(w http.ResponseWriter, r *http.Request) {
	bucket := chi.URLParam(r, "bucket")
	h.logger.Debug("DeleteObjects request", zap.String("bucket", bucket))

	// Parse request body
	var deleteReq models.DeleteObjectsRequest
	if err := xml.NewDecoder(r.Body).Decode(&deleteReq); err != nil {
		h.logger.Error("failed to parse delete objects request", zap.Error(err))
		s3Err := &models.S3Error{
			Code:    models.MalformedXML,
			Message: "The XML you provided was not well-formed or did not validate against our published schema",
		}
		h.writeErrorResponse(w, s3Err)
		return
	}

	result := models.DeleteResult{}

	// Process each object deletion
	for _, obj := range deleteReq.Objects {
		var err error
		if obj.VersionID != "" {
			err = h.backend.DeleteObjectVersion(r.Context(), bucket, obj.Key, obj.VersionID)
		} else {
			err = h.backend.DeleteObject(r.Context(), bucket, obj.Key)
		}

		if err != nil {
			h.logger.Warn("failed to delete object in batch",
				zap.String("bucket", bucket),
				zap.String("key", obj.Key),
				zap.Error(err))

			result.Error = append(result.Error, models.ErrorResult{
				Key:       obj.Key,
				VersionID: obj.VersionID,
				Code:      "InternalError",
				Message:   err.Error(),
			})
		} else {
			if !deleteReq.Quiet {
				result.Deleted = append(result.Deleted, models.DeletedObject{
					Key:       obj.Key,
					VersionID: obj.VersionID,
				})
			}

			// Invalidate cache if enabled
			if h.cacheManager != nil {
				cacheKey := generateCacheKey(bucket, obj.Key)
				_ = h.cacheManager.InvalidateObject(cacheKey)
			}
		}
	}

	h.stats.RecordDeleteObject(true)
	if h.telMgr != nil {
		h.telMgr.RecordS3Request(r.Context(), "DeleteObjects", true, "")
	}

	w.Header().Set("Content-Type", "application/xml")
	w.WriteHeader(http.StatusOK)
	xmlData, _ := xml.Marshal(result)
	_, _ = w.Write(xmlData)
}
