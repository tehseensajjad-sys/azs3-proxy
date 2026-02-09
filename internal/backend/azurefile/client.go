package azurefile

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azfile/directory"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azfile/file"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azfile/fileerror"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azfile/service"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azfile/share"
	"go.uber.org/zap"

	"github.com/vibhansa-msft/azs3-proxy/internal/backend"
	backendcommon "github.com/vibhansa-msft/azs3-proxy/internal/backend/common"
	"github.com/vibhansa-msft/azs3-proxy/internal/config"
	"github.com/vibhansa-msft/azs3-proxy/internal/telemetry"
)

const (
	// memoryUploadThreshold defines the maximum size for in-memory uploads.
	// Files smaller than this are buffered in memory; larger files use streaming.
	// 256MB is safe for typical workloads and avoids excessive memory allocation.
	memoryUploadThreshold = 256 * 1024 * 1024 // 256MB

	// maxPartSize defines the maximum size for a single multipart upload part.
	// This prevents memory exhaustion from extremely large parts.
	maxPartSize = 256 * 1024 * 1024 // 256MB per part

	// maxMultipartUploadSize defines the maximum total size for a multipart upload
	// that we'll buffer in memory. Uploads larger than this will fail with an error.
	maxMultipartUploadSize = 5 * 1024 * 1024 * 1024 // 5GB
)

// AzureFileBackend implements the storage backend interface using Azure Files.
// It handles all file operations, multipart uploads, and share management.
// Maps S3 concepts to Azure Files: buckets→shares, objects→files.
type AzureFileBackend struct {
	client                *service.Client                     // Azure Files Service client
	shareClients          sync.Map                            // Cache for share clients (shareName -> *share.Client)
	multipartUploads      map[string]*MultipartUploadMetadata // Ongoing multipart uploads
	multipartUploadsMutex sync.RWMutex                        // Thread-safe access to multipart uploads
	versionedShares       map[string]bool                     // Tracks shares with versioning enabled (simulated)
	versionedSharesMutex  sync.RWMutex                        // Thread-safe access to versioning state
	logger                *zap.Logger                         // Logger for backend operations
	telMgr                *telemetry.Manager                  // Telemetry manager for metrics
	limiter               *backendcommon.BandwidthLimiter     // Optional bandwidth limiter (Azure-facing only)
}

// Options configures Azure Files backend construction.
type Options struct {
	AuthConfig      *config.AzureAuthConfig
	CapMbpsRead     float64
	CapMbpsWrite    float64
	CapMbpsCombined float64
	Logger          *zap.Logger
	Telemetry       *telemetry.Manager
}

// MultipartUploadMetadata stores metadata about an active multipart upload.
// Azure Files doesn't have native multipart upload like Blob Storage,
// so we buffer parts in memory and upload the complete file when finalized.
type MultipartUploadMetadata struct {
	UploadID   string         // Unique ID for this multipart upload
	ShareName  string         // Share name in Azure Files
	FilePath   string         // File path in Azure Files
	Parts      map[int][]byte // Maps part number to part data (buffered in memory)
	PartOrder  []int          // Ordered list of part numbers
	Initiated  time.Time      // When the multipart upload was initiated
	FileClient *file.Client   // Client for file operations
	Mutex      sync.Mutex     // Mutex to protect Parts and PartOrder
}

var (
	// clientCache stores Azure Files service clients to avoid recreating them for the same credentials.
	// This is a global cache shared across all AzureFileBackend instances to maximize reuse.
	// Access is protected by clientCacheMutex to ensure thread-safety.
	// Note: This pattern matches the azureblob backend implementation for consistency.
	clientCache      = make(map[string]*service.Client)
	clientCacheMutex sync.RWMutex
)

func (af *AzureFileBackend) wrapDownload(rc io.ReadCloser) io.ReadCloser {
	if rc == nil {
		return nil
	}
	if af.limiter == nil {
		return rc
	}
	return af.limiter.WrapDownload(rc)
}

func (af *AzureFileBackend) wrapUpload(r io.Reader) io.Reader {
	if r == nil {
		return nil
	}
	if af.limiter == nil {
		return r
	}
	return af.limiter.WrapUpload(r)
}

// UpdateCaps adjusts bandwidth caps at runtime. When no limiter exists and
// any cap is positive, a new limiter is created; non-positive caps disable
// their respective buckets.
func (af *AzureFileBackend) UpdateCaps(capReadMbps, capWriteMbps, capCombinedMbps float64) {
	if af == nil {
		return
	}

	if af.limiter == nil {
		af.limiter = backendcommon.NewBandwidthLimiter(capReadMbps, capWriteMbps, capCombinedMbps)
		return
	}

	af.limiter.UpdateCaps(capReadMbps, capWriteMbps, capCombinedMbps)
}

// NewAzureFileBackendWithAuth creates an Azure Files backend using flexible authentication.
// Supports account key and SAS token authentication methods.
// Logs authentication method and storage account for audit/debugging purposes.
func NewAzureFileBackendWithAuth(opts Options) (*AzureFileBackend, error) {
	if opts.AuthConfig == nil {
		return nil, fmt.Errorf("auth config is required")
	}
	if opts.Logger == nil {
		opts.Logger = zap.NewNop()
	}

	ctx := context.Background()

	opts.Logger.Info("initializing Azure Files backend",
		zap.String("storage_account", opts.AuthConfig.StorageAccountName),
		zap.String("auth_mode", opts.AuthConfig.Mode.String()))

	cacheKey := backendcommon.BuildCacheKey(opts.AuthConfig)
	clientCacheMutex.RLock()
	client, ok := clientCache[cacheKey]
	clientCacheMutex.RUnlock()

	if ok {
		opts.Logger.Info("using cached azure files client",
			zap.String("storage_account", opts.AuthConfig.StorageAccountName),
			zap.String("auth_mode", opts.AuthConfig.Mode.String()))
	} else {
		var err error
		telemetryEnabled := opts.Telemetry != nil && opts.Telemetry.IsEnabled()
		client, err = BuildServiceClientFromCredential(ctx, opts.AuthConfig, opts.Logger, telemetryEnabled)
		if err != nil {
			opts.Logger.Error("failed to build azure files client",
				zap.Error(err),
				zap.String("storage_account", opts.AuthConfig.StorageAccountName),
				zap.String("auth_mode", opts.AuthConfig.Mode.String()))

			return nil, fmt.Errorf("failed to build azure files client: %w", err)
		}

		clientCacheMutex.Lock()
		clientCache[cacheKey] = client
		clientCacheMutex.Unlock()
	}

	opts.Logger.Info("Azure Files backend initialized successfully",
		zap.String("storage_account", opts.AuthConfig.StorageAccountName),
		zap.String("auth_mode", opts.AuthConfig.Mode.String()))

	return &AzureFileBackend{
		client:           client,
		multipartUploads: make(map[string]*MultipartUploadMetadata),
		versionedShares:  make(map[string]bool),
		logger:           opts.Logger,
		telMgr:           opts.Telemetry,
		limiter:          backendcommon.NewBandwidthLimiter(opts.CapMbpsRead, opts.CapMbpsWrite, opts.CapMbpsCombined),
	}, nil
}

// getShareClient returns a share client, using cache when possible.
// We cache these using sync.Map to avoid lock contention on hot paths.
func (af *AzureFileBackend) getShareClient(shareName string) *share.Client {
	if client, ok := af.shareClients.Load(shareName); ok {
		return client.(*share.Client)
	}

	newClient := af.client.NewShareClient(shareName)
	actual, _ := af.shareClients.LoadOrStore(shareName, newClient)
	return actual.(*share.Client)
}

// createParentDirectories creates all parent directories for a file path.
// Azure Files requires parent directories to exist before creating files.
func (af *AzureFileBackend) createParentDirectories(ctx context.Context, shareClient *share.Client, filePath string) error {
	// Split path into directory components
	dir := path.Dir(filePath)
	if dir == "." || dir == "/" || dir == "" {
		return nil // No parent directories needed
	}

	// Build directory path incrementally
	parts := strings.Split(strings.Trim(dir, "/"), "/")
	currentPath := ""

	for _, part := range parts {
		if part == "" {
			continue
		}

		if currentPath == "" {
			currentPath = part
		} else {
			currentPath = currentPath + "/" + part
		}

		// Create directory (ignore if already exists)
		dirClient := shareClient.NewDirectoryClient(currentPath)
		_, err := dirClient.Create(ctx, nil)
		if err != nil && !fileerror.HasCode(err, fileerror.ResourceAlreadyExists) {
			return fmt.Errorf("failed to create directory %s: %w", currentPath, err)
		}
	}

	return nil
}

// ListBuckets returns all shares in the Azure Files account.
// Shares in Azure Files correspond to buckets in S3.
func (af *AzureFileBackend) ListBuckets(ctx context.Context) (buckets []string, err error) {
	defer func() { af.recordAzureRequest(ctx, "ListBuckets", err) }()

	pager := af.client.NewListSharesPager(nil)

	// Iterate through all pages of share results
	for pager.More() {
		resp, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("list shares failed: %w", err)
		}

		// Extract share names from the response
		if resp.Shares != nil {
			for _, s := range resp.Shares {
				if s.Name != nil {
					buckets = append(buckets, *s.Name)
				}
			}
		}
	}
	return buckets, nil
}

// HeadBucket checks if a share exists in Azure Files.
// It uses GetProperties to verify existence and accessibility.
// Returns true if the share exists, false if it doesn't (404), and error for other failures.
func (af *AzureFileBackend) HeadBucket(ctx context.Context, bucketName string) (exists bool, err error) {
	defer func() { af.recordAzureRequest(ctx, "HeadBucket", err) }()

	shareClient := af.getShareClient(bucketName)
	_, err = shareClient.GetProperties(ctx, nil)
	if err != nil {
		// Check if error is 404 Not Found
		if fileerror.HasCode(err, fileerror.ShareNotFound) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// CreateBucket creates a new share in Azure Files.
func (af *AzureFileBackend) CreateBucket(ctx context.Context, bucketName string) (err error) {
	defer func() { af.recordAzureRequest(ctx, "CreateBucket", err) }()

	_, err = af.getShareClient(bucketName).Create(ctx, nil)
	if err != nil {
		return fmt.Errorf("create share failed: %w", err)
	}
	return nil
}

// DeleteBucket deletes a share from Azure Files.
func (af *AzureFileBackend) DeleteBucket(ctx context.Context, bucketName string) (err error) {
	defer func() { af.recordAzureRequest(ctx, "DeleteBucket", err) }()

	_, err = af.getShareClient(bucketName).Delete(ctx, nil)
	if err != nil {
		return fmt.Errorf("delete share failed: %w", err)
	}
	return nil
}

// PutObject uploads a file to Azure Files.
// Creates parent directories as needed.
func (af *AzureFileBackend) PutObject(ctx context.Context, bucketName, objectKey string, size int64, data io.Reader) (err error) {
	defer func() { af.recordAzureRequest(ctx, "PutObject", err) }()

	shareClient := af.getShareClient(bucketName)

	// Create parent directories if needed
	if err := af.createParentDirectories(ctx, shareClient, objectKey); err != nil {
		return fmt.Errorf("failed to create parent directories: %w", err)
	}

	// Get file client
	fileClient := shareClient.NewRootDirectoryClient().NewFileClient(objectKey)

	// Create the file with the specified size
	_, err = fileClient.Create(ctx, size, nil)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}

	useLimiter := af.limiter != nil

	// Upload data to the file
	// For small files, read into memory and use UploadBuffer unless limiter is enabled
	if !useLimiter && size > 0 && size <= memoryUploadThreshold {
		// Read entire body into memory
		buf := make([]byte, size)
		_, err := io.ReadFull(data, buf)
		if err != nil {
			return fmt.Errorf("failed to read body into memory: %w", err)
		}

		// Upload buffer to file
		err = fileClient.UploadBuffer(ctx, buf, nil)
		if err != nil {
			return fmt.Errorf("upload buffer failed: %w", err)
		}
		return nil
	}

	if useLimiter {
		data = af.wrapUpload(data)
	}

	// For larger files or when limiter is enabled, use UploadStream
	err = fileClient.UploadStream(ctx, data, nil)
	if err != nil {
		return fmt.Errorf("upload stream failed: %w", err)
	}

	return nil
}

// CopyObject copies a file from source to destination within Azure Files.
func (af *AzureFileBackend) CopyObject(ctx context.Context, srcBucket, srcKey, destBucket, destKey string) (err error) {
	defer func() { af.recordAzureRequest(ctx, "CopyObject", err) }()

	// Get source object
	srcInfo, err := af.GetObject(ctx, srcBucket, srcKey)
	if err != nil {
		return fmt.Errorf("failed to get source object for copy: %w", err)
	}
	defer func() { _ = srcInfo.Body.Close() }()

	// Put to destination
	err = af.PutObject(ctx, destBucket, destKey, srcInfo.Size, srcInfo.Body)
	if err != nil {
		return fmt.Errorf("failed to upload destination object for copy: %w", err)
	}

	return nil
}

// DeleteObject deletes a file from Azure Files.
func (af *AzureFileBackend) DeleteObject(ctx context.Context, bucketName, objectKey string) (err error) {
	defer func() { af.recordAzureRequest(ctx, "DeleteObject", err) }()

	shareClient := af.getShareClient(bucketName)
	fileClient := shareClient.NewRootDirectoryClient().NewFileClient(objectKey)

	_, err = fileClient.Delete(ctx, nil)
	if err != nil {
		// S3 idempotency: If file is not found, return success
		if fileerror.HasCode(err, fileerror.ResourceNotFound) {
			return nil
		}
		return fmt.Errorf("delete file failed: %w", err)
	}
	return nil
}

// HeadObject checks if a file exists and returns its metadata.
func (af *AzureFileBackend) HeadObject(ctx context.Context, bucketName, objectKey string) (exists bool, size int64, lastModified time.Time, err error) {
	defer func() { af.recordAzureRequest(ctx, "HeadObject", err) }()

	// Defensive: empty objectKey indicates a bucket-level HEAD
	if strings.TrimSpace(objectKey) == "" {
		return false, 0, time.Time{}, nil
	}

	shareClient := af.getShareClient(bucketName)
	fileClient := shareClient.NewRootDirectoryClient().NewFileClient(objectKey)

	props, err := fileClient.GetProperties(ctx, nil)
	if err != nil {
		// Map Azure SDK responses that indicate the file doesn't exist
		if fileerror.HasCode(err, fileerror.ResourceNotFound) || fileerror.HasCode(err, fileerror.ShareNotFound) {
			return false, 0, time.Time{}, nil
		}
		return false, 0, time.Time{}, fmt.Errorf("get file properties failed: %w", err)
	}

	var sz int64
	if props.ContentLength != nil {
		sz = *props.ContentLength
	}
	var lm time.Time
	if props.LastModified != nil {
		lm = props.LastModified.UTC()
	}
	return true, sz, lm, nil
}

// listAllObjects returns all files in a share with optional prefix filtering.
// Caller is responsible for telemetry recording to avoid double counting.
func (af *AzureFileBackend) listAllObjects(ctx context.Context, bucketName, prefix string) (objects []string, err error) {
	shareClient := af.getShareClient(bucketName)
	rootDirClient := shareClient.NewRootDirectoryClient()

	// Recursively list files in the share
	var listFiles func(dirClient *directory.Client, dirPath string) error
	listFiles = func(dirClient *directory.Client, dirPath string) error {
		pager := dirClient.NewListFilesAndDirectoriesPager(nil)

		for pager.More() {
			resp, err := pager.NextPage(ctx)
			if err != nil {
				return fmt.Errorf("list files and directories failed: %w", err)
			}

			// Process files
			if resp.Segment.Files != nil {
				for _, fileItem := range resp.Segment.Files {
					if fileItem.Name != nil {
						filePath := *fileItem.Name
						if dirPath != "" {
							filePath = dirPath + "/" + filePath
						}
						// Apply prefix filter
						if prefix == "" || strings.HasPrefix(filePath, prefix) {
							objects = append(objects, filePath)
						}
					}
				}
			}

			// Process directories recursively
			if resp.Segment.Directories != nil {
				for _, dirItem := range resp.Segment.Directories {
					if dirItem.Name != nil {
						subDirPath := *dirItem.Name
						if dirPath != "" {
							subDirPath = dirPath + "/" + subDirPath
						}
						// Only recurse if prefix matches or could match
						if prefix == "" || strings.HasPrefix(prefix, subDirPath) || strings.HasPrefix(subDirPath, prefix) {
							subDirClient := shareClient.NewDirectoryClient(subDirPath)
							if err := listFiles(subDirClient, subDirPath); err != nil {
								return err
							}
						}
					}
				}
			}
		}
		return nil
	}

	err = listFiles(rootDirClient, "")
	if err != nil {
		return nil, err
	}

	return objects, nil
}

// ListObjects lists all files in a share with optional prefix filter.
func (af *AzureFileBackend) ListObjects(ctx context.Context, bucketName, prefix string) (objects []string, err error) {
	defer func() { af.recordAzureRequest(ctx, "ListObjects", err) }()
	return af.listAllObjects(ctx, bucketName, prefix)
}

// ListObjectsV2 returns a single page of files with pagination tokens encoded as offsets.
func (af *AzureFileBackend) ListObjectsV2(ctx context.Context, bucketName, prefix, continuationToken string, maxResults int32) (objects []string, nextContinuationToken string, err error) {
	defer func() { af.recordAzureRequest(ctx, "ListObjects", err) }()

	allObjects, err := af.listAllObjects(ctx, bucketName, prefix)
	if err != nil {
		return nil, "", err
	}

	if maxResults <= 0 {
		maxResults = 1000
	}

	offset := 0
	if continuationToken != "" {
		if parsed, parseErr := strconv.Atoi(continuationToken); parseErr == nil && parsed >= 0 && parsed < len(allObjects) {
			offset = parsed
		}
	}

	end := offset + int(maxResults)
	if end > len(allObjects) {
		end = len(allObjects)
	}

	objects = allObjects[offset:end]
	if end < len(allObjects) {
		nextContinuationToken = strconv.Itoa(end)
	}

	return objects, nextContinuationToken, nil
}

// GetObject returns file data and metadata for S3 compatibility.
func (af *AzureFileBackend) GetObject(ctx context.Context, bucketName, objectKey string) (backend.ObjectInfo, error) {
	var info backend.ObjectInfo

	shareClient := af.getShareClient(bucketName)
	fileClient := shareClient.NewRootDirectoryClient().NewFileClient(objectKey)

	// Get file properties first
	props, err := fileClient.GetProperties(ctx, nil)
	if err != nil {
		return info, fmt.Errorf("get file properties failed: %w", err)
	}

	if props.LastModified != nil {
		info.LastModified = props.LastModified.UTC()
	} else {
		info.LastModified = time.Now().UTC()
	}
	if props.ContentLength != nil {
		info.Size = *props.ContentLength
	}

	// Download file
	// For all files, use DownloadStream for efficient streaming
	resp, err := fileClient.DownloadStream(ctx, nil)
	if err != nil {
		return info, fmt.Errorf("download file failed: %w", err)
	}
	info.Body = af.wrapDownload(resp.Body)

	return info, nil
}

// GetObjectRange returns a byte range for the file, clamping to the file size when needed.
func (af *AzureFileBackend) GetObjectRange(ctx context.Context, bucketName, objectKey string, offset, length int64) (info backend.ObjectInfo, err error) {
	defer func() { af.recordAzureRequest(ctx, "GetObjectRange", err) }()

	shareClient := af.getShareClient(bucketName)
	fileClient := shareClient.NewRootDirectoryClient().NewFileClient(objectKey)

	props, err := fileClient.GetProperties(ctx, nil)
	if err != nil {
		return info, fmt.Errorf("get file properties failed: %w", err)
	}

	if props.ContentLength != nil {
		info.Size = *props.ContentLength
	}
	if props.LastModified != nil {
		info.LastModified = props.LastModified.UTC()
	} else {
		info.LastModified = time.Now().UTC()
	}

	if info.Size <= 0 {
		return info, fmt.Errorf("object has zero length")
	}
	if offset < 0 {
		offset = 0
	}
	if offset >= info.Size {
		return info, fmt.Errorf("range start beyond size")
	}
	if length <= 0 {
		length = info.Size - offset
	}
	end := offset + length - 1
	if end >= info.Size {
		end = info.Size - 1
	}
	count := end - offset + 1

	resp, err := fileClient.DownloadStream(ctx, &file.DownloadStreamOptions{
		Range: file.HTTPRange{Offset: offset, Count: count},
	})
	if err != nil {
		return info, fmt.Errorf("download range failed: %w", err)
	}

	info.Body = af.wrapDownload(resp.Body)
	return info, nil
}

// InitiateMultipartUpload initiates a new multipart upload.
// Azure Files doesn't have native multipart upload, so we simulate it by buffering parts.
func (af *AzureFileBackend) InitiateMultipartUpload(ctx context.Context, bucketName, objectKey string) (string, error) {
	uploadID := fmt.Sprintf("%s-%s-%d", bucketName, objectKey, time.Now().UnixNano())

	shareClient := af.getShareClient(bucketName)
	fileClient := shareClient.NewRootDirectoryClient().NewFileClient(objectKey)

	af.multipartUploadsMutex.Lock()
	defer af.multipartUploadsMutex.Unlock()

	af.multipartUploads[uploadID] = &MultipartUploadMetadata{
		UploadID:   uploadID,
		ShareName:  bucketName,
		FilePath:   objectKey,
		Parts:      make(map[int][]byte),
		PartOrder:  []int{},
		Initiated:  time.Now(),
		FileClient: fileClient,
	}

	return uploadID, nil
}

// UploadPart uploads a single part of a multipart upload.
// Parts are buffered in memory until CompleteMultipartUpload is called.
// NOTE: Large uploads may cause memory pressure as parts are buffered in memory.
func (af *AzureFileBackend) UploadPart(ctx context.Context, bucketName, objectKey, uploadID string, partNumber int, size int64, data io.Reader) (etag string, err error) {
	defer func() { af.recordAzureRequest(ctx, "UploadPart", err) }()

	// Validate part size to prevent memory exhaustion
	if size > maxPartSize {
		return "", fmt.Errorf("part size %d exceeds maximum allowed size %d", size, maxPartSize)
	}

	// Validate upload exists first
	af.multipartUploadsMutex.RLock()
	upload, exists := af.multipartUploads[uploadID]
	af.multipartUploadsMutex.RUnlock()

	if !exists {
		return "", fmt.Errorf("no such upload: %s", uploadID)
	}

	if upload.ShareName != bucketName || upload.FilePath != objectKey {
		return "", fmt.Errorf("upload ID does not match bucket/key")
	}

	// Calculate current total size to prevent exceeding limits
	upload.Mutex.Lock()
	var currentSize int64
	for _, partData := range upload.Parts {
		currentSize += int64(len(partData))
	}
	upload.Mutex.Unlock()

	if currentSize+size > maxMultipartUploadSize {
		return "", fmt.Errorf("multipart upload exceeds maximum size %d (current: %d, part: %d)", maxMultipartUploadSize, currentSize, size)
	}

	// Read part data into memory with size limit
	var partData []byte
	if size > 0 {
		// Pre-allocate buffer when size is known
		partData = make([]byte, size)
		_, err = io.ReadFull(data, partData)
		if err != nil {
			return "", fmt.Errorf("failed to read part data: %w", err)
		}
	} else {
		// Size unknown, use limited reader to prevent unbounded growth
		limitedReader := io.LimitReader(data, maxPartSize+1)
		partData, err = io.ReadAll(limitedReader)
		if err != nil {
			return "", fmt.Errorf("failed to read part data: %w", err)
		}
		if int64(len(partData)) > maxPartSize {
			return "", fmt.Errorf("part exceeds maximum size %d", maxPartSize)
		}
	}

	// Store part data
	upload.Mutex.Lock()
	defer upload.Mutex.Unlock()

	// Check if part already exists and update it
	_, alreadyExists := upload.Parts[partNumber]
	upload.Parts[partNumber] = partData

	// Only add to PartOrder if it's a new part
	if !alreadyExists {
		upload.PartOrder = append(upload.PartOrder, partNumber)
	}

	// Generate ETag for the part
	etag = fmt.Sprintf("\"%s-%d\"", uploadID, partNumber)

	return etag, nil
}

// CompleteMultipartUpload completes a multipart upload by assembling all parts.
func (af *AzureFileBackend) CompleteMultipartUpload(ctx context.Context, bucketName, objectKey, uploadID string, partETags map[int]string) (etag string, err error) {
	defer func() { af.recordAzureRequest(ctx, "CompleteMultipartUpload", err) }()

	af.multipartUploadsMutex.RLock()
	upload, exists := af.multipartUploads[uploadID]
	af.multipartUploadsMutex.RUnlock()

	if !exists {
		return "", fmt.Errorf("no such upload: %s", uploadID)
	}

	if upload.ShareName != bucketName || upload.FilePath != objectKey {
		return "", fmt.Errorf("upload ID does not match bucket/key")
	}

	upload.Mutex.Lock()
	if len(upload.Parts) == 0 {
		upload.Mutex.Unlock()
		return "", fmt.Errorf("no parts uploaded")
	}

	// Assemble parts in ascending numerical order (not upload order)
	// Sort part numbers to ensure correct assembly
	partNumbers := make([]int, 0, len(upload.Parts))
	for partNum := range upload.Parts {
		partNumbers = append(partNumbers, partNum)
	}

	// Sort in ascending order
	for i := 0; i < len(partNumbers); i++ {
		for j := i + 1; j < len(partNumbers); j++ {
			if partNumbers[i] > partNumbers[j] {
				partNumbers[i], partNumbers[j] = partNumbers[j], partNumbers[i]
			}
		}
	}

	// Assemble parts in correct order
	var totalSize int64
	var assembledData bytes.Buffer

	for _, partNum := range partNumbers {
		partData := upload.Parts[partNum]
		totalSize += int64(len(partData))
		assembledData.Write(partData)
	}
	upload.Mutex.Unlock()

	// Create parent directories if needed
	shareClient := af.getShareClient(bucketName)
	if err := af.createParentDirectories(ctx, shareClient, objectKey); err != nil {
		return "", fmt.Errorf("failed to create parent directories: %w", err)
	}

	// Upload the complete file
	err = af.PutObject(ctx, bucketName, objectKey, totalSize, &assembledData)
	if err != nil {
		return "", fmt.Errorf("failed to upload complete file: %w", err)
	}

	// Clean up the multipart upload metadata
	af.multipartUploadsMutex.Lock()
	defer af.multipartUploadsMutex.Unlock()
	delete(af.multipartUploads, uploadID)

	return fmt.Sprintf("\"%s\"", uploadID), nil
}

// AbortMultipartUpload aborts an ongoing multipart upload.
func (af *AzureFileBackend) AbortMultipartUpload(ctx context.Context, bucketName, objectKey, uploadID string) error {
	af.multipartUploadsMutex.Lock()
	defer af.multipartUploadsMutex.Unlock()

	upload, exists := af.multipartUploads[uploadID]
	if !exists {
		return fmt.Errorf("no such upload: %s", uploadID)
	}

	if upload.ShareName != bucketName || upload.FilePath != objectKey {
		return fmt.Errorf("upload ID does not match bucket/key")
	}

	delete(af.multipartUploads, uploadID)
	return nil
}

// ListParts lists all parts of an ongoing multipart upload.
func (af *AzureFileBackend) ListParts(ctx context.Context, bucketName, objectKey, uploadID string) ([]interface{}, error) {
	af.multipartUploadsMutex.RLock()
	upload, exists := af.multipartUploads[uploadID]
	af.multipartUploadsMutex.RUnlock()

	if !exists {
		return nil, fmt.Errorf("no such upload: %s", uploadID)
	}

	if upload.ShareName != bucketName || upload.FilePath != objectKey {
		return nil, fmt.Errorf("upload ID does not match bucket/key")
	}

	upload.Mutex.Lock()
	defer upload.Mutex.Unlock()

	var parts []interface{}
	for _, partNum := range upload.PartOrder {
		partData := upload.Parts[partNum]
		parts = append(parts, map[string]interface{}{
			"PartNumber": partNum,
			"ETag":       fmt.Sprintf("\"%s-%d\"", uploadID, partNum),
			"Size":       int64(len(partData)),
		})
	}

	return parts, nil
}

// ListMultipartUploads lists all ongoing multipart uploads in a share.
func (af *AzureFileBackend) ListMultipartUploads(ctx context.Context, bucketName string) ([]interface{}, error) {
	af.multipartUploadsMutex.RLock()
	defer af.multipartUploadsMutex.RUnlock()

	var uploads []interface{}
	for _, upload := range af.multipartUploads {
		if upload.ShareName == bucketName {
			uploads = append(uploads, map[string]interface{}{
				"Key":       upload.FilePath,
				"UploadID":  upload.UploadID,
				"Initiated": upload.Initiated.Format(time.RFC3339),
			})
		}
	}

	return uploads, nil
}

// EnableVersioning enables versioning for a share.
// Note: Azure Files doesn't support versioning natively like Blob Storage.
// This is a simulated implementation that just tracks the state.
func (af *AzureFileBackend) EnableVersioning(ctx context.Context, bucketName string) error {
	af.versionedSharesMutex.Lock()
	defer af.versionedSharesMutex.Unlock()

	// Check if share exists
	shareClient := af.getShareClient(bucketName)
	_, err := shareClient.GetProperties(ctx, nil)
	if err != nil {
		return fmt.Errorf("share %s does not exist: %w", bucketName, err)
	}

	af.versionedShares[bucketName] = true
	return nil
}

// GetVersioning returns whether versioning is enabled for a share.
func (af *AzureFileBackend) GetVersioning(ctx context.Context, bucketName string) (bool, error) {
	af.versionedSharesMutex.RLock()
	defer af.versionedSharesMutex.RUnlock()

	return af.versionedShares[bucketName], nil
}

// ListObjectVersions lists all versions of objects in a share.
// Azure Files doesn't support versioning natively, so this returns current files only.
func (af *AzureFileBackend) ListObjectVersions(ctx context.Context, bucketName, prefix string) ([]interface{}, error) {
	// Get all objects
	objects, err := af.ListObjects(ctx, bucketName, prefix)
	if err != nil {
		return nil, err
	}

	var versions []interface{}
	for _, objKey := range objects {
		// Get object metadata
		exists, size, lastModified, err := af.HeadObject(ctx, bucketName, objKey)
		if err != nil || !exists {
			continue
		}

		version := backend.ObjectVersion{
			Key:       objKey,
			VersionID: "null", // Azure Files doesn't support versioning
			ETag:      fmt.Sprintf("\"%s\"", objKey),
			Size:      size,
			Modified:  lastModified.Format(time.RFC3339),
			IsLatest:  true,
		}
		versions = append(versions, version)
	}

	return versions, nil
}

// GetObjectVersion retrieves a specific version of an object.
// Azure Files doesn't support versioning, so this just returns the current object.
func (af *AzureFileBackend) GetObjectVersion(ctx context.Context, bucketName, objectKey, versionID string) (backend.ObjectInfo, error) {
	// Azure Files doesn't support versioning, so just return the current object
	return af.GetObject(ctx, bucketName, objectKey)
}

// DeleteObjectVersion deletes a specific version of an object.
// Azure Files doesn't support versioning, so this just deletes the current object.
func (af *AzureFileBackend) DeleteObjectVersion(ctx context.Context, bucketName, objectKey, versionID string) error {
	// Azure Files doesn't support versioning, so just delete the current object
	return af.DeleteObject(ctx, bucketName, objectKey)
}

// Close clears the global client cache and releases resources.
func (af *AzureFileBackend) Close() error {
	clientCacheMutex.Lock()
	defer clientCacheMutex.Unlock()

	// Clear the cache
	for k := range clientCache {
		delete(clientCache, k)
	}

	return nil
}

// recordAzureRequest records metrics and logs for Azure operations.
func (af *AzureFileBackend) recordAzureRequest(ctx context.Context, operation string, err error) {
	success := err == nil
	errorType := ""
	if err != nil {
		errorType = "Unknown"
		// Try to extract Azure error code
		type hasErrorCode interface {
			ErrorCode() string
		}
		if fileerr, ok := err.(hasErrorCode); ok {
			errorType = fileerr.ErrorCode()
		}
	}

	if af.telMgr != nil {
		af.telMgr.RecordAzureRequest(ctx, operation, success, errorType)
	}

	if af.logger != nil {
		reqID := backendcommon.RequestIDFromContext(ctx)
		fields := []zap.Field{
			zap.String("operation", operation),
			zap.String("direction", "outbound"),
		}
		if reqID != "" {
			fields = append(fields, zap.String("req_id", reqID))
		}
		if success {
			af.logger.Debug("azure files request successful", fields...)
		} else {
			fields = append(fields,
				zap.Error(err),
				zap.String("error_type", errorType),
			)
			af.logger.Error("azure files request failed", fields...)
		}
	}
}
