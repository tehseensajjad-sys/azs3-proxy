package azureblob

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/container"
	"go.uber.org/zap"

	"github.com/vibhansa-msft/s3-azure-proxy/internal/config"
)

type AzureBlobBackend struct {
	client *azblob.Client
}

func NewAzureBlobBackend(connectionString string) (*AzureBlobBackend, error) {
	client, err := azblob.NewClientFromConnectionString(connectionString, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create azure client: %w", err)
	}
	return &AzureBlobBackend{client: client}, nil
}

// NewAzureBlobBackendWithAuth creates an Azure Blob backend using flexible authentication
// Supports multiple auth modes: account key, SAS, MSI, SPN, federated token, Azure CLI
func NewAzureBlobBackendWithAuth(authConfig *config.AzureAuthConfig, logger *zap.Logger) (*AzureBlobBackend, error) {
	ctx := context.Background()
	client, err := BuildClientFromCredential(ctx, authConfig, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to build azure blob client: %w", err)
	}
	return &AzureBlobBackend{client: client}, nil
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
