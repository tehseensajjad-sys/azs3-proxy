package backend

import (
	"context"
	"io"
)

// StorageBackend defines the interface for storage backend implementations
// that translate S3 operations to cloud-specific operations
type StorageBackend interface {
	// Bucket operations
	ListBuckets(ctx context.Context) ([]string, error)
	CreateBucket(ctx context.Context, bucketName string) error
	DeleteBucket(ctx context.Context, bucketName string) error

	// Object operations
	PutObject(ctx context.Context, bucketName, objectKey string, data io.Reader) error
	GetObject(ctx context.Context, bucketName, objectKey string) (io.ReadCloser, error)
	DeleteObject(ctx context.Context, bucketName, objectKey string) error
	HeadObject(ctx context.Context, bucketName, objectKey string) (bool, error)
	ListObjects(ctx context.Context, bucketName, prefix string) ([]string, error)

	// Multipart upload operations
	InitiateMultipartUpload(ctx context.Context, bucketName, objectKey string) (uploadID string, err error)
	UploadPart(ctx context.Context, bucketName, objectKey, uploadID string, partNumber int, data io.Reader) (etag string, err error)
	CompleteMultipartUpload(ctx context.Context, bucketName, objectKey, uploadID string, partETags map[int]string) (etag string, err error)
	AbortMultipartUpload(ctx context.Context, bucketName, objectKey, uploadID string) error
	ListParts(ctx context.Context, bucketName, objectKey, uploadID string) ([]interface{}, error)
	ListMultipartUploads(ctx context.Context, bucketName string) ([]interface{}, error)

	// Versioning operations
	EnableVersioning(ctx context.Context, bucketName string) error
	GetVersioning(ctx context.Context, bucketName string) (enabled bool, err error)
	ListObjectVersions(ctx context.Context, bucketName, prefix string) ([]interface{}, error)
	GetObjectVersion(ctx context.Context, bucketName, objectKey, versionID string) (io.ReadCloser, error)
	DeleteObjectVersion(ctx context.Context, bucketName, objectKey, versionID string) error
}

// Part represents information about an uploaded part
type Part struct {
	PartNumber int
	ETag       string
	Size       int64
}

// UploadInfo represents information about an ongoing multipart upload
type UploadInfo struct {
	Key       string
	UploadID  string
	Initiated string
}

// ObjectVersion represents a specific version of an object
type ObjectVersion struct {
	Key       string
	VersionID string
	ETag      string
	Size      int64
	Modified  string
	IsLatest  bool
}
