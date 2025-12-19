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

	"github.com/vibhansa-msft/s3-azure-proxy/internal/backend"
	"github.com/vibhansa-msft/s3-azure-proxy/internal/config"
)

// readSeekCloser wraps bytes.Reader to implement io.ReadSeekCloser interface.
// Used for objects stored in memory that need to be read multiple times or seeked.
type readSeekCloser struct {
	*bytes.Reader
}

// Close is a no-op for in-memory byte readers.
func (r *readSeekCloser) Close() error {
	return nil
}

// AzureBlobBackend implements the storage backend interface using Azure Blob Storage.
// It handles all blob operations, multipart uploads, and bucket versioning.
type AzureBlobBackend struct {
	client                *azblob.Client                      // Azure Blob Storage client
	multipartUploads      map[string]*MultipartUploadMetadata // Ongoing multipart uploads
	multipartUploadsMutex sync.RWMutex                        // Thread-safe access to multipart uploads
	versionedBuckets      map[string]bool                     // Tracks buckets with versioning enabled
	versionedBucketsMutex sync.RWMutex                        // Thread-safe access to versioning state
}

// MultipartUploadMetadata stores metadata about an active multipart upload.
// Azure Blob Storage uses a staging blocks approach: individual parts are uploaded as blocks,
// then combined using PutBlockList to create the final blob.
type MultipartUploadMetadata struct {
	UploadID        string            // Unique ID for this multipart upload
	BucketName      string            // Container name in Azure
	ObjectKey       string            // Blob name in Azure
	BlockIDs        []string          // Ordered list of block IDs (corresponding to part numbers)
	PartETagMap     map[int]string    // Maps part number to block ID (etag equivalent)
	Initiated       time.Time         // When the multipart upload was initiated
	BlockBlobClient *blockblob.Client // Client for block blob operations
}

// NewAzureBlobBackend creates an Azure Blob backend using a connection string.
// This is a basic constructor; NewAzureBlobBackendWithAuth is preferred for flexible authentication.
func NewAzureBlobBackend(connectionString string) (*AzureBlobBackend, error) {
	client, err := azblob.NewClientFromConnectionString(connectionString, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create azure client: %w", err)
	}

	return &AzureBlobBackend{
		client:           client,
		multipartUploads: make(map[string]*MultipartUploadMetadata),
		versionedBuckets: make(map[string]bool),
	}, nil
}

var (
	clientCache      = make(map[string]*azblob.Client)
	clientCacheMutex sync.RWMutex
)

func getClientCacheKey(cfg *config.AzureAuthConfig) string {
	return fmt.Sprintf("%s|%s|%s|%s|%s|%s|%s",
		cfg.Mode,
		cfg.StorageAccountName,
		cfg.AccountKey,
		cfg.SASToken,
		cfg.MSIClientID,
		cfg.SPNClientID,
		cfg.FederatedClientID,
	)
}

// NewAzureBlobBackendWithAuth creates an Azure Blob backend using flexible authentication.
// Supports six authentication methods: account key, SAS, MSI, SPN, federated token, and Azure CLI.
// Logs authentication method and storage account for audit/debugging purposes.
func NewAzureBlobBackendWithAuth(authConfig *config.AzureAuthConfig, logger *zap.Logger) (*AzureBlobBackend, error) {
	ctx := context.Background()

	// Log initialization with auth method for debugging and audit
	logger.Info("initializing Azure Blob backend",
		zap.String("storage_account", authConfig.StorageAccountName),
		zap.String("auth_mode", authConfig.Mode.String()))

	// Check cache first
	cacheKey := getClientCacheKey(authConfig)
	clientCacheMutex.RLock()
	client, ok := clientCache[cacheKey]
	clientCacheMutex.RUnlock()

	if ok {
		logger.Info("using cached azure blob client",
			zap.String("storage_account", authConfig.StorageAccountName),
			zap.String("auth_mode", authConfig.Mode.String()))
	} else {
		var err error
		// Build Azure client using the appropriate authentication credential
		client, err = BuildClientFromCredential(ctx, authConfig, logger)
		if err != nil {
			logger.Error("failed to build azure blob client",
				zap.Error(err),
				zap.String("storage_account", authConfig.StorageAccountName),
				zap.String("auth_mode", authConfig.Mode.String()))

			return nil, fmt.Errorf("failed to build azure blob client: %w", err)
		}

		// Update cache
		clientCacheMutex.Lock()
		clientCache[cacheKey] = client
		clientCacheMutex.Unlock()
	}

	// Log successful initialization
	logger.Info("Azure Blob backend initialized successfully",
		zap.String("storage_account", authConfig.StorageAccountName),
		zap.String("auth_mode", authConfig.Mode.String()))

	return &AzureBlobBackend{
		client:           client,
		multipartUploads: make(map[string]*MultipartUploadMetadata),
		versionedBuckets: make(map[string]bool),
	}, nil
}

// ListBuckets returns all containers in the Azure Blob Storage account.
// Containers in Azure Blob Storage correspond to buckets in S3.
func (ab *AzureBlobBackend) ListBuckets(ctx context.Context) ([]string, error) {
	var buckets []string
	pager := ab.client.NewListContainersPager(nil)

	// Iterate through all pages of container results
	for pager.More() {
		resp, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("list containers failed: %w", err)
		}

		// Extract container names from the response
		if resp.ContainerItems != nil {
			for _, c := range resp.ContainerItems {
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

func (ab *AzureBlobBackend) CopyObject(ctx context.Context, srcBucket, srcKey, destBucket, destKey string) error {
	// Naive implementation: Download from source and upload to destination.
	// This avoids the complexity of SAS token generation for StartCopyFromURL
	// when we don't have direct access to the account key here.
	// TODO: Optimize using StartCopyFromURL if possible.

	// 1. Get source object stream
	srcResp, err := ab.GetObject(ctx, srcBucket, srcKey)
	if err != nil {
		return fmt.Errorf("failed to open source object for copy: %w", err)
	}
	defer func() { _ = srcResp.Close() }()

	// 2. Put to destination
	err = ab.PutObject(ctx, destBucket, destKey, srcResp)
	if err != nil {
		return fmt.Errorf("failed to upload destination object for copy: %w", err)
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

// EnableVersioning enables versioning for a bucket
func (ab *AzureBlobBackend) EnableVersioning(ctx context.Context, bucketName string) error {
	ab.versionedBucketsMutex.Lock()
	defer ab.versionedBucketsMutex.Unlock()

	// Check if bucket exists
	containerClient := ab.client.ServiceClient().NewContainerClient(bucketName)
	_, err := containerClient.GetProperties(ctx, nil)
	if err != nil {
		return fmt.Errorf("bucket %s does not exist: %w", bucketName, err)
	}

	ab.versionedBuckets[bucketName] = true
	return nil
}

// GetVersioning returns whether versioning is enabled for a bucket
func (ab *AzureBlobBackend) GetVersioning(ctx context.Context, bucketName string) (bool, error) {
	ab.versionedBucketsMutex.RLock()
	defer ab.versionedBucketsMutex.RUnlock()

	return ab.versionedBuckets[bucketName], nil
}

// ListObjectVersions lists all versions of all objects in a bucket
func (ab *AzureBlobBackend) ListObjectVersions(ctx context.Context, bucketName, prefix string) ([]interface{}, error) {
	containerClient := ab.client.ServiceClient().NewContainerClient(bucketName)

	var versions []interface{}
	pager := containerClient.NewListBlobsFlatPager(&container.ListBlobsFlatOptions{
		Prefix: &prefix,
	})

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list blob versions: %w", err)
		}

		for _, blobItem := range page.Segment.BlobItems {
			versionID := ""
			if blobItem.VersionID != nil {
				versionID = *blobItem.VersionID
			}
			version := backend.ObjectVersion{
				Key:       *blobItem.Name,
				VersionID: versionID,
				ETag:      string(*blobItem.Properties.ETag),
				Size:      *blobItem.Properties.ContentLength,
				Modified:  blobItem.Properties.LastModified.Format(time.RFC3339),
				IsLatest:  true, // Mark as latest in versioned listing
			}
			versions = append(versions, version)
		}
	}

	return versions, nil
}

// GetObjectVersion retrieves a specific version of an object
func (ab *AzureBlobBackend) GetObjectVersion(ctx context.Context, bucketName, objectKey, versionID string) (io.ReadCloser, error) {
	containerClient := ab.client.ServiceClient().NewContainerClient(bucketName)
	blobClient := containerClient.NewBlockBlobClient(objectKey)

	// In Azure Blob Storage, versionID is a snapshot ID or version timestamp
	// We'll attempt to use it as-is for downloading a specific version
	resp, err := blobClient.DownloadStream(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get object version: %w", err)
	}

	return resp.Body, nil
}

// DeleteObjectVersion deletes a specific version of an object
func (ab *AzureBlobBackend) DeleteObjectVersion(ctx context.Context, bucketName, objectKey, versionID string) error {
	containerClient := ab.client.ServiceClient().NewContainerClient(bucketName)
	blobClient := containerClient.NewBlockBlobClient(objectKey)

	// In Azure Blob Storage, versionID is managed through blob versioning
	// For now, we'll delete the blob without specific version support
	_, err := blobClient.Delete(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to delete object version: %w", err)
	}

	return nil
}

// Close clears the global client cache and releases resources.
func (ab *AzureBlobBackend) Close() error {
	clientCacheMutex.Lock()
	defer clientCacheMutex.Unlock()

	// Clear the map
	for k := range clientCache {
		delete(clientCache, k)
	}

	return nil
}
