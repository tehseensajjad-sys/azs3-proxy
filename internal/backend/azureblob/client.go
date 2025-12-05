package azureblob

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blockblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/container"
	"go.uber.org/zap"

	"github.com/vibhansa-msft/s3-azure-proxy/internal/config"
)

type readSeekCloser struct {
	*bytes.Reader
}

func (r *readSeekCloser) Close() error {
	return nil
}

type AzureBlobBackend struct {
	client                *azblob.Client
	multipartUploads      map[string]*MultipartUploadMetadata
	multipartUploadsMutex sync.RWMutex
}

// MultipartUploadMetadata stores metadata about an ongoing multipart upload
// Uses Azure's staging blocks approach - parts are uploaded as blocks and finalized with PutBlockList
type MultipartUploadMetadata struct {
	UploadID        string
	BucketName      string
	ObjectKey       string
	BlockIDs        []string       // ordered list of block IDs
	PartETagMap     map[int]string // part number -> etag (block ID)
	Initiated       time.Time
	BlockBlobClient *blockblob.Client
}

func NewAzureBlobBackend(connectionString string) (*AzureBlobBackend, error) {
	client, err := azblob.NewClientFromConnectionString(connectionString, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create azure client: %w", err)
	}
	return &AzureBlobBackend{
		client:           client,
		multipartUploads: make(map[string]*MultipartUploadMetadata),
	}, nil
}

// NewAzureBlobBackendWithAuth creates an Azure Blob backend using flexible authentication
// Supports multiple auth modes: account key, SAS, MSI, SPN, federated token, Azure CLI
func NewAzureBlobBackendWithAuth(authConfig *config.AzureAuthConfig, logger *zap.Logger) (*AzureBlobBackend, error) {
	ctx := context.Background()
	client, err := BuildClientFromCredential(ctx, authConfig, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to build azure blob client: %w", err)
	}
	return &AzureBlobBackend{
		client:           client,
		multipartUploads: make(map[string]*MultipartUploadMetadata),
	}, nil
}

func (ab *AzureBlobBackend) ListBuckets(ctx context.Context) ([]string, error) {
	var buckets []string
	pager := ab.client.NewListContainersPager(nil)

	for pager.More() {
		resp, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("list containers failed: %w", err)
		}
		if resp.ListContainersSegmentResponse.ContainerItems != nil {
			for _, c := range resp.ListContainersSegmentResponse.ContainerItems {
				if c.Name != nil {
					buckets = append(buckets, *c.Name)
				}
			}
		}
	}
	return buckets, nil
}

func (ab *AzureBlobBackend) CreateBucket(ctx context.Context, bucketName string) error {
	_, err := ab.client.ServiceClient().NewContainerClient(bucketName).Create(ctx, nil)
	if err != nil {
		return fmt.Errorf("create container failed: %w", err)
	}
	return nil
}

func (ab *AzureBlobBackend) DeleteBucket(ctx context.Context, bucketName string) error {
	_, err := ab.client.ServiceClient().NewContainerClient(bucketName).Delete(ctx, nil)
	if err != nil {
		return fmt.Errorf("delete container failed: %w", err)
	}
	return nil
}

func (ab *AzureBlobBackend) PutObject(ctx context.Context, bucketName, objectKey string, data io.Reader) error {
	_, err := ab.client.ServiceClient().NewContainerClient(bucketName).NewBlockBlobClient(objectKey).UploadStream(ctx, data, nil)
	if err != nil {
		return fmt.Errorf("upload blob failed: %w", err)
	}
	return nil
}

func (ab *AzureBlobBackend) GetObject(ctx context.Context, bucketName, objectKey string) (io.ReadCloser, error) {
	resp, err := ab.client.ServiceClient().NewContainerClient(bucketName).NewBlockBlobClient(objectKey).DownloadStream(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("download blob failed: %w", err)
	}
	return resp.Body, nil
}

func (ab *AzureBlobBackend) DeleteObject(ctx context.Context, bucketName, objectKey string) error {
	_, err := ab.client.ServiceClient().NewContainerClient(bucketName).NewBlockBlobClient(objectKey).Delete(ctx, nil)
	if err != nil {
		return fmt.Errorf("delete blob failed: %w", err)
	}
	return nil
}

func (ab *AzureBlobBackend) HeadObject(ctx context.Context, bucketName, objectKey string) (bool, error) {
	_, err := ab.client.ServiceClient().NewContainerClient(bucketName).NewBlockBlobClient(objectKey).GetProperties(ctx, nil)
	if err != nil {
		if strings.Contains(err.Error(), "404") || strings.Contains(err.Error(), "BlobNotFound") {
			return false, nil
		}
		return false, fmt.Errorf("get blob properties failed: %w", err)
	}
	return true, nil
}

func (ab *AzureBlobBackend) ListObjects(ctx context.Context, bucketName, prefix string) ([]string, error) {
	var objects []string
	containerClient := ab.client.ServiceClient().NewContainerClient(bucketName)
	options := &container.ListBlobsFlatOptions{Prefix: &prefix}

	pager := containerClient.NewListBlobsFlatPager(options)
	for pager.More() {
		resp, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("list blobs failed: %w", err)
		}
		if resp.Segment != nil && resp.Segment.BlobItems != nil {
			for _, blob := range resp.Segment.BlobItems {
				if blob.Name != nil {
					objects = append(objects, *blob.Name)
				}
			}
		}
	}
	return objects, nil
}

// InitiateMultipartUpload initiates a new multipart upload
func (ab *AzureBlobBackend) InitiateMultipartUpload(ctx context.Context, bucketName, objectKey string) (string, error) {
	uploadID := fmt.Sprintf("%s-%s-%d", bucketName, objectKey, time.Now().UnixNano())

	// Create block blob client for this upload
	blockBlobClient := ab.client.ServiceClient().NewContainerClient(bucketName).NewBlockBlobClient(objectKey)

	ab.multipartUploadsMutex.Lock()
	defer ab.multipartUploadsMutex.Unlock()

	ab.multipartUploads[uploadID] = &MultipartUploadMetadata{
		UploadID:        uploadID,
		BucketName:      bucketName,
		ObjectKey:       objectKey,
		BlockIDs:        []string{},
		PartETagMap:     make(map[int]string),
		Initiated:       time.Now(),
		BlockBlobClient: blockBlobClient,
	}

	return uploadID, nil
} // UploadPart uploads a single part of a multipart upload
// Thread-safe: holds lock throughout to prevent concurrent block loss
func (ab *AzureBlobBackend) UploadPart(ctx context.Context, bucketName, objectKey, uploadID string, partNumber int, data io.Reader) (string, error) {
	// Validate upload exists first (before reading data)
	ab.multipartUploadsMutex.RLock()
	upload, exists := ab.multipartUploads[uploadID]
	ab.multipartUploadsMutex.RUnlock()

	if !exists {
		return "", fmt.Errorf("no such upload: %s", uploadID)
	}

	if upload.BucketName != bucketName || upload.ObjectKey != objectKey {
		return "", fmt.Errorf("upload ID does not match bucket/key")
	}

	// Read part data (outside lock to avoid blocking)
	partData, err := io.ReadAll(data)
	if err != nil {
		return "", fmt.Errorf("failed to read part data: %w", err)
	}

	// Create a block ID based on part number (base64 encoded)
	blockID := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%010d", partNumber)))

	// Hold lock for the critical section: stage block + update metadata atomically
	ab.multipartUploadsMutex.Lock()
	defer ab.multipartUploadsMutex.Unlock()

	// Re-verify upload still exists after releasing lock
	upload, exists = ab.multipartUploads[uploadID]
	if !exists {
		return "", fmt.Errorf("no such upload: %s", uploadID)
	}

	// Stage the block in Azure using readSeekCloser wrapper
	_, err = upload.BlockBlobClient.StageBlock(ctx, blockID, &readSeekCloser{bytes.NewReader(partData)}, nil)
	if err != nil {
		return "", fmt.Errorf("failed to stage block: %w", err)
	}

	// Update upload metadata with block ID and part etag
	// This is now atomic with the StageBlock call
	upload.BlockIDs = append(upload.BlockIDs, blockID)
	etag := fmt.Sprintf("\"%s\"", blockID)
	upload.PartETagMap[partNumber] = etag

	return etag, nil
}

// CompleteMultipartUpload completes a multipart upload by combining all parts
func (ab *AzureBlobBackend) CompleteMultipartUpload(ctx context.Context, bucketName, objectKey, uploadID string, partETags map[int]string) (string, error) {
	ab.multipartUploadsMutex.RLock()
	upload, exists := ab.multipartUploads[uploadID]
	ab.multipartUploadsMutex.RUnlock()

	if !exists {
		return "", fmt.Errorf("no such upload: %s", uploadID)
	}

	if upload.BucketName != bucketName || upload.ObjectKey != objectKey {
		return "", fmt.Errorf("upload ID does not match bucket/key")
	}

	if len(upload.BlockIDs) == 0 {
		return "", fmt.Errorf("no parts uploaded")
	}

	// Finalize the multipart upload by committing the staged blocks
	_, err := upload.BlockBlobClient.CommitBlockList(ctx, upload.BlockIDs, nil)
	if err != nil {
		return "", fmt.Errorf("failed to commit block list: %w", err)
	}

	// Clean up the multipart upload metadata
	ab.multipartUploadsMutex.Lock()
	defer ab.multipartUploadsMutex.Unlock()
	delete(ab.multipartUploads, uploadID)

	// Return combined ETag (S3-style with count and dash)
	etag := fmt.Sprintf("\"%s-%d\"", uploadID[:8], len(upload.BlockIDs)) // S3-style combined ETag with part count

	return etag, nil
}

// AbortMultipartUpload aborts an ongoing multipart upload
func (ab *AzureBlobBackend) AbortMultipartUpload(ctx context.Context, bucketName, objectKey, uploadID string) error {
	ab.multipartUploadsMutex.Lock()
	defer ab.multipartUploadsMutex.Unlock()

	upload, exists := ab.multipartUploads[uploadID]
	if !exists {
		return fmt.Errorf("no such upload: %s", uploadID)
	}

	if upload.BucketName != bucketName || upload.ObjectKey != objectKey {
		return fmt.Errorf("upload ID does not match bucket/key")
	}

	delete(ab.multipartUploads, uploadID)
	return nil
}

// ListParts lists all parts of an ongoing multipart upload
func (ab *AzureBlobBackend) ListParts(ctx context.Context, bucketName, objectKey, uploadID string) ([]interface{}, error) {
	ab.multipartUploadsMutex.RLock()
	upload, exists := ab.multipartUploads[uploadID]
	ab.multipartUploadsMutex.RUnlock()

	if !exists {
		return nil, fmt.Errorf("no such upload: %s", uploadID)
	}

	if upload.BucketName != bucketName || upload.ObjectKey != objectKey {
		return nil, fmt.Errorf("upload ID does not match bucket/key")
	}

	var parts []interface{}
	for i, blockID := range upload.BlockIDs {
		partNum := i + 1
		parts = append(parts, map[string]interface{}{
			"PartNumber": partNum,
			"ETag":       fmt.Sprintf("\"%s\"", blockID),
			"Size":       int64(0), // Size is not tracked in current implementation
		})
	}

	return parts, nil
}

// ListMultipartUploads lists all ongoing multipart uploads in a bucket
func (ab *AzureBlobBackend) ListMultipartUploads(ctx context.Context, bucketName string) ([]interface{}, error) {
	ab.multipartUploadsMutex.RLock()
	defer ab.multipartUploadsMutex.RUnlock()

	var uploads []interface{}
	for _, upload := range ab.multipartUploads {
		if upload.BucketName == bucketName {
			uploads = append(uploads, map[string]interface{}{
				"Key":       upload.ObjectKey,
				"UploadID":  upload.UploadID,
				"Initiated": upload.Initiated.Format(time.RFC3339),
			})
		}
	}

	return uploads, nil
}
