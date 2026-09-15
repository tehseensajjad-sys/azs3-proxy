package backend

import (
	"context"
	"io"
	"time"
)

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

// ObjectListItem holds the metadata returned for each object in list operations.
type ObjectListItem struct {
	Key          string
	Size         int64
	LastModified time.Time
	ETag         string
}

// ObjectInfo holds object data and metadata for S3 responses
// (add more fields as needed for S3 compatibility)
type ObjectInfo struct {
	Body         io.ReadCloser
	LastModified time.Time
	Size         int64
	// ETag is the real S3-style content-hash ETag (quoted hex MD5), if the
	// backend has one available. Empty if unavailable (e.g. legacy objects
	// written before the backend started persisting a content hash), in
	// which case callers should fall back to a placeholder ETag.
	ETag string
}

// StorageBackend defines the interface for storage backend implementations
// that translate S3 operations to cloud-specific operations
type StorageBackend interface {
	// Bucket operations
	ListBuckets(ctx context.Context) ([]string, error)
	CreateBucket(ctx context.Context, bucketName string) error
	DeleteBucket(ctx context.Context, bucketName string) error
	HeadBucket(ctx context.Context, bucketName string) (bool, error)

	// Object operations
	// PutObject returns the real content-hash ETag (S3-style, quoted hex MD5)
	// for the uploaded object.
	PutObject(ctx context.Context, bucketName, objectKey string, size int64, data io.Reader) (etag string, err error)
	// CopyObject returns the real content-hash ETag (S3-style, quoted hex MD5)
	// of the copied object.
	CopyObject(ctx context.Context, srcBucket, srcKey, destBucket, destKey string) (etag string, err error)
	GetObject(ctx context.Context, bucketName, objectKey string) (ObjectInfo, error)
	// GetObjectRange returns a byte range [offset, offset+length-1] for the object.
	// length must be >0. Implementations should clamp to object size and return the
	// available data if the range exceeds the end of the object.
	GetObjectRange(ctx context.Context, bucketName, objectKey string, offset, length int64) (ObjectInfo, error)
	DeleteObject(ctx context.Context, bucketName, objectKey string) error
	// HeadObject returns whether the object exists. When exists is true,
	// size contains the object size in bytes, lastModified contains the
	// object's last modified time (UTC), and etag contains the real S3-style
	// content-hash ETag if available (empty otherwise). If the object does
	// not exist, exists will be false and size/lastModified/etag will be
	// zero values.
	HeadObject(ctx context.Context, bucketName, objectKey string) (exists bool, size int64, lastModified time.Time, etag string, err error)
	ListObjects(ctx context.Context, bucketName, prefix string) ([]ObjectListItem, error)
	ListObjectsV2(ctx context.Context, bucketName, prefix, continuationToken string, maxResults int32) (objects []ObjectListItem, nextContinuationToken string, err error)

	// Multipart upload operations
	InitiateMultipartUpload(ctx context.Context, bucketName, objectKey string) (uploadID string, err error)
	UploadPart(ctx context.Context, bucketName, objectKey, uploadID string, partNumber int, size int64, data io.Reader) (etag string, err error)
	CompleteMultipartUpload(ctx context.Context, bucketName, objectKey, uploadID string, partETags map[int]string) (etag string, err error)
	AbortMultipartUpload(ctx context.Context, bucketName, objectKey, uploadID string) error
	ListParts(ctx context.Context, bucketName, objectKey, uploadID string) ([]interface{}, error)
	ListMultipartUploads(ctx context.Context, bucketName string) ([]interface{}, error)

	// Versioning operations
	EnableVersioning(ctx context.Context, bucketName string) error
	GetVersioning(ctx context.Context, bucketName string) (enabled bool, err error)
	ListObjectVersions(ctx context.Context, bucketName, prefix string) ([]interface{}, error)
	GetObjectVersion(ctx context.Context, bucketName, objectKey, versionID string) (ObjectInfo, error)
	DeleteObjectVersion(ctx context.Context, bucketName, objectKey, versionID string) error

	// Lifecycle operations
	Close() error
}
