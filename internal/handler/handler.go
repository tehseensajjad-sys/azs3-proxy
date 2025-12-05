package handler

import (
	"encoding/xml"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/vibhansa-msft/s3-azure-proxy/internal/backend"
	"github.com/vibhansa-msft/s3-azure-proxy/internal/models"
)

// S3Handler handles S3 API requests
type S3Handler struct {
	backend backend.StorageBackend
	logger  *zap.Logger
}

// NewS3Handler creates a new S3 handler
func NewS3Handler(backend backend.StorageBackend, logger *zap.Logger) *S3Handler {
	return &S3Handler{
		backend: backend,
		logger:  logger,
	}
}

// Helper function to write error response
func (h *S3Handler) writeErrorResponse(w http.ResponseWriter, err *models.S3Error) {
	w.Header().Set("Content-Type", "application/xml")
	w.WriteHeader(err.HTTPStatus())

	resp := models.ErrorResponse{
		Code:      string(err.Code),
		Message:   err.Message,
		Resource:  err.Resource,
		RequestID: err.RequestID,
	}

	xmlData, _ := xml.Marshal(resp)
	w.Write(xmlData)
} // Helper function to extract bucket and object key from request
func extractBucketAndKey(r *http.Request) (string, string) {
	bucket := chi.URLParam(r, "bucket")
	key := strings.TrimPrefix(chi.URLParam(r, "*"), "/")
	return bucket, key
}

// Bucket Operations

// ListBucketsHandler handles GET / (ListBuckets)
func (h *S3Handler) ListBucketsHandler(w http.ResponseWriter, r *http.Request) {
	h.logger.Debug("ListBuckets request")

	buckets, err := h.backend.ListBuckets(r.Context())
	if err != nil {
		h.logger.Error("failed to list buckets", zap.Error(err))
		s3Err := &models.S3Error{
			Code:    models.InternalError,
			Message: "Failed to list buckets",
		}
		h.writeErrorResponse(w, s3Err)
		return
	}

	bucketList := make([]models.Bucket, len(buckets))
	for i, name := range buckets {
		bucketList[i] = models.Bucket{
			Name:         name,
			CreationDate: time.Now().UTC().Format(time.RFC3339),
		}
	}

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
	w.Write(xmlData)
}

// CreateBucketHandler handles PUT /{bucket}
func (h *S3Handler) CreateBucketHandler(w http.ResponseWriter, r *http.Request) {
	bucket := chi.URLParam(r, "bucket")
	h.logger.Debug("CreateBucket request", zap.String("bucket", bucket))

	err := h.backend.CreateBucket(r.Context(), bucket)
	if err != nil {
		h.logger.Error("failed to create bucket", zap.Error(err), zap.String("bucket", bucket))
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
}

// DeleteBucketHandler handles DELETE /{bucket}
func (h *S3Handler) DeleteBucketHandler(w http.ResponseWriter, r *http.Request) {
	bucket := chi.URLParam(r, "bucket")
	h.logger.Debug("DeleteBucket request", zap.String("bucket", bucket))

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

	w.WriteHeader(http.StatusNoContent)
}

// ListObjectsV2Handler handles GET /{bucket} with list-type=2
func (h *S3Handler) ListObjectsV2Handler(w http.ResponseWriter, r *http.Request) {
	bucket := chi.URLParam(r, "bucket")
	prefix := r.URL.Query().Get("prefix")
	h.logger.Debug("ListObjectsV2 request", zap.String("bucket", bucket), zap.String("prefix", prefix))

	objects, err := h.backend.ListObjects(r.Context(), bucket, prefix)
	if err != nil {
		h.logger.Error("failed to list objects", zap.Error(err), zap.String("bucket", bucket))
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
	w.Write(xmlData)
}

// Object Operations

// PutObjectHandler handles PUT /{bucket}/{key}
func (h *S3Handler) PutObjectHandler(w http.ResponseWriter, r *http.Request) {
	bucket, key := extractBucketAndKey(r)
	h.logger.Debug("PutObject request", zap.String("bucket", bucket), zap.String("key", key))

	if key == "" {
		// This is a bucket operation, not an object operation
		h.CreateBucketHandler(w, r)
		return
	}

	err := h.backend.PutObject(r.Context(), bucket, key, r.Body)
	if err != nil {
		h.logger.Error("failed to put object", zap.Error(err), zap.String("bucket", bucket), zap.String("key", key))
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

	w.Header().Set("ETag", "\"0\"")
	w.WriteHeader(http.StatusOK)
}

// GetObjectHandler handles GET /{bucket}/{key}
func (h *S3Handler) GetObjectHandler(w http.ResponseWriter, r *http.Request) {
	bucket, key := extractBucketAndKey(r)
	h.logger.Debug("GetObject request", zap.String("bucket", bucket), zap.String("key", key))

	if key == "" {
		// This is a list operation, not a get object operation
		h.ListObjectsV2Handler(w, r)
		return
	}

	body, err := h.backend.GetObject(r.Context(), bucket, key)
	if err != nil {
		h.logger.Error("failed to get object", zap.Error(err), zap.String("bucket", bucket), zap.String("key", key))
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
	defer body.Close()

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("ETag", "\"0\"")
	w.WriteHeader(http.StatusOK)
	io.Copy(w, body)
}

// HeadObjectHandler handles HEAD /{bucket}/{key}
func (h *S3Handler) HeadObjectHandler(w http.ResponseWriter, r *http.Request) {
	bucket, key := extractBucketAndKey(r)
	h.logger.Debug("HeadObject request", zap.String("bucket", bucket), zap.String("key", key))

	exists, err := h.backend.HeadObject(r.Context(), bucket, key)
	if err != nil {
		h.logger.Error("failed to head object", zap.Error(err), zap.String("bucket", bucket), zap.String("key", key))
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

	w.WriteHeader(http.StatusNoContent)
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
	w.Write(xmlData)
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
		s3Err := &models.S3Error{
			Code:    models.InternalError,
			Message: err.Error(),
		}
		h.writeErrorResponse(w, s3Err)
		return
	}

	w.Header().Set("ETag", etag)
	w.WriteHeader(http.StatusOK)
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
	w.Write(xmlData)
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
		s3Err := &models.S3Error{
			Code:    models.InternalError,
			Message: err.Error(),
		}
		h.writeErrorResponse(w, s3Err)
		return
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
	w.Write(xmlData)
}

// ListMultipartUploadsHandler handles GET /{bucket}?uploads
func (h *S3Handler) ListMultipartUploadsHandler(w http.ResponseWriter, r *http.Request) {
	bucket := chi.URLParam(r, "bucket")
	h.logger.Debug("ListMultipartUploads request", zap.String("bucket", bucket))

	uploads, err := h.backend.ListMultipartUploads(r.Context(), bucket)
	if err != nil {
		h.logger.Error("failed to list multipart uploads", zap.Error(err))
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

	resp := models.ListMultipartUploadsResponse{
		Bucket:      bucket,
		MaxUploads:  1000,
		IsTruncated: false,
		Uploads:     uploadList,
	}
	xmlData, _ := xml.Marshal(resp)
	w.Write(xmlData)
}
