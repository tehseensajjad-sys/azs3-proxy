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
}
