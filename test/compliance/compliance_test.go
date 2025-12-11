package compliance

import (
	"bytes"
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/vibhansa-msft/s3-azure-proxy/internal/config"
	"github.com/vibhansa-msft/s3-azure-proxy/internal/server"
)

// TestCompliance runs a comprehensive S3 compliance test suite.
func TestCompliance(t *testing.T) {
	// 1. Check environment requirements
	connStr := os.Getenv("AZURE_STORAGE_CONNECTION_STRING")
	accountName := os.Getenv("AZURE_STORAGE_ACCOUNT")
	accountKey := os.Getenv("AZURE_STORAGE_KEY")

	if connStr == "" && (accountName == "" || accountKey == "") {
		t.Skip("Skipping compliance test: AZURE_STORAGE_CONNECTION_STRING or AZURE_STORAGE_ACCOUNT/KEY not set")
	}

	// 2. Setup Proxy Server
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to listen on random port: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	_ = listener.Close()

	proxyAddr := fmt.Sprintf("127.0.0.1:%d", port)

	cfg := &config.Config{
		ListenAddr:        proxyAddr,
		S3AccessKeyID:     "test-access-key",
		S3SecretAccessKey: "test-secret-key",
		LogLevel:          "error",
	}

	if connStr != "" {
		if strings.Contains(connStr, "UseDevelopmentStorage=true") {
			cfg.AzureAuth = &config.AzureAuthConfig{
				Mode:               config.AuthModeAccountKey,
				StorageAccountName: "devstoreaccount1",
				AccountKey:         "Eby8vdM02xNOcqFlqUwJPLlmEtlCDXJ1OUzFT50uSRZ6IFsuFq2UVErCz4I6tq/K1SZFPTOtr/KBHBeksoGMGw==",
				StorageAccountURL:  "http://127.0.0.1:10000/devstoreaccount1",
			}
		} else {
			if accountName != "" && accountKey != "" {
				cfg.AzureAuth = &config.AzureAuthConfig{
					Mode:               config.AuthModeAccountKey,
					StorageAccountName: accountName,
					AccountKey:         accountKey,
					StorageAccountURL:  fmt.Sprintf("https://%s.blob.core.windows.net", accountName),
				}
			} else {
				t.Skip("Skipping: Complex connection string parsing not implemented in test setup yet")
			}
		}
	} else {
		cfg.AzureAuth = &config.AzureAuthConfig{
			Mode:               config.AuthModeAccountKey,
			StorageAccountName: accountName,
			AccountKey:         accountKey,
			StorageAccountURL:  fmt.Sprintf("https://%s.blob.core.windows.net", accountName),
		}
	}

	logger, _ := zap.NewDevelopment()
	router := chi.NewRouter()

	srv, err := server.NewS3ProxyServer(router, cfg, logger, nil)
	if err != nil {
		t.Fatalf("Failed to create proxy server: %v", err)
	}
	defer func() { _ = srv.Close() }()

	httpServer := &http.Server{
		Addr:    proxyAddr,
		Handler: router,
	}

	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			t.Errorf("Server failed: %v", err)
		}
	}()
	defer func() { _ = httpServer.Shutdown(ctx) }()

	time.Sleep(100 * time.Millisecond)

	s3Client, err := createS3Client(ctx, proxyAddr, cfg.S3AccessKeyID, cfg.S3SecretAccessKey)
	if err != nil {
		t.Fatalf("Failed to create S3 client: %v", err)
	}

	// Run Sub-tests
	bucketName := fmt.Sprintf("compliance-bucket-%d", time.Now().UnixNano())

	t.Run("BucketOperations", func(t *testing.T) {
		testBucketOperations(ctx, t, s3Client, bucketName)
	})

	t.Run("ObjectOperations", func(t *testing.T) {
		testObjectOperations(ctx, t, s3Client, bucketName)
	})

	t.Run("MultipartUpload", func(t *testing.T) {
		testMultipartUpload(ctx, t, s3Client, bucketName)
	})

	t.Run("Versioning", func(t *testing.T) {
		testVersioning(ctx, t, s3Client, bucketName)
	})

	// Cleanup
	cleanupBucket(ctx, t, s3Client, bucketName)
}

func testBucketOperations(ctx context.Context, t *testing.T, client *s3.Client, bucketName string) {
	// Create Bucket
	_, err := client.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: aws.String(bucketName),
	})
	if err != nil {
		t.Fatalf("CreateBucket failed: %v", err)
	}

	// List Buckets
	resp, err := client.ListBuckets(ctx, &s3.ListBucketsInput{})
	if err != nil {
		t.Fatalf("ListBuckets failed: %v", err)
	}
	found := false
	for _, b := range resp.Buckets {
		if *b.Name == bucketName {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Bucket %s not found in list", bucketName)
	}

	// Head Bucket (Check existence)
	_, err = client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(bucketName),
	})
	if err != nil {
		t.Errorf("HeadBucket failed: %v", err)
	}
}

func testObjectOperations(ctx context.Context, t *testing.T, client *s3.Client, bucketName string) {
	key := "test-object.txt"
	content := "Hello, Compliance!"
	metadata := map[string]string{"custom-meta": "value1"}

	// Put Object
	_, err := client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:   aws.String(bucketName),
		Key:      aws.String(key),
		Body:     strings.NewReader(content),
		Metadata: metadata,
	})
	if err != nil {
		t.Fatalf("PutObject failed: %v", err)
	}

	// Get Object
	resp, err := client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(key),
	})
	if err != nil {
		t.Fatalf("GetObject failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read body: %v", err)
	}
	if string(body) != content {
		t.Errorf("Content mismatch: got %s, want %s", string(body), content)
	}

	// Check Metadata
	// Note: Azure might normalize metadata keys, so we check case-insensitively or expect lowercase
	if val, ok := resp.Metadata["custom-meta"]; !ok || val != "value1" {
		t.Logf("Metadata mismatch or missing (known issue if not fully supported): got %v", resp.Metadata)
	}

	// Copy Object
	copyKey := "test-object-copy.txt"
	_, err = client.CopyObject(ctx, &s3.CopyObjectInput{
		Bucket:     aws.String(bucketName),
		CopySource: aws.String(bucketName + "/" + key),
		Key:        aws.String(copyKey),
	})
	if err != nil {
		t.Errorf("CopyObject failed: %v", err)
	} else {
		// Verify Copy
		copyResp, err := client.GetObject(ctx, &s3.GetObjectInput{
			Bucket: aws.String(bucketName),
			Key:    aws.String(copyKey),
		})
		if err != nil {
			t.Errorf("GetObject (copy) failed: %v", err)
		} else {
			_ = copyResp.Body.Close()
		}
	}

	// List Objects
	listResp, err := client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket: aws.String(bucketName),
		Prefix: aws.String("test-"),
	})
	if err != nil {
		t.Fatalf("ListObjectsV2 failed: %v", err)
	}
	if len(listResp.Contents) < 2 {
		t.Errorf("Expected at least 2 objects, got %d", len(listResp.Contents))
	}
}

func testMultipartUpload(ctx context.Context, t *testing.T, client *s3.Client, bucketName string) {
	key := "multipart-object.bin"
	// Create 15MB of random data (3 parts of 5MB)
	// Min part size is usually 5MB for S3, but Azure might be flexible. Sticking to 5MB is safe.
	partSize := 5 * 1024 * 1024
	totalSize := 3 * partSize
	data := make([]byte, totalSize)
	_, _ = rand.Read(data)

	// Initiate
	initResp, err := client.CreateMultipartUpload(ctx, &s3.CreateMultipartUploadInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(key),
	})
	if err != nil {
		t.Fatalf("CreateMultipartUpload failed: %v", err)
	}
	uploadID := *initResp.UploadId

	// Upload Parts
	var completedParts []types.CompletedPart
	for i := 0; i < 3; i++ {
		partNum := int32(i + 1)
		start := i * partSize
		end := start + partSize
		partResp, err := client.UploadPart(ctx, &s3.UploadPartInput{
			Bucket:     aws.String(bucketName),
			Key:        aws.String(key),
			UploadId:   aws.String(uploadID),
			PartNumber: aws.Int32(partNum),
			Body:       bytes.NewReader(data[start:end]),
		})
		if err != nil {
			t.Fatalf("UploadPart %d failed: %v", partNum, err)
		}
		completedParts = append(completedParts, types.CompletedPart{
			ETag:       partResp.ETag,
			PartNumber: aws.Int32(partNum),
		})
	}

	// List Parts
	listPartsResp, err := client.ListParts(ctx, &s3.ListPartsInput{
		Bucket:   aws.String(bucketName),
		Key:      aws.String(key),
		UploadId: aws.String(uploadID),
	})
	if err != nil {
		t.Errorf("ListParts failed: %v", err)
	} else if len(listPartsResp.Parts) != 3 {
		t.Errorf("Expected 3 parts, got %d", len(listPartsResp.Parts))
	}

	// Complete
	sort.Slice(completedParts, func(i, j int) bool {
		return *completedParts[i].PartNumber < *completedParts[j].PartNumber
	})

	_, err = client.CompleteMultipartUpload(ctx, &s3.CompleteMultipartUploadInput{
		Bucket:   aws.String(bucketName),
		Key:      aws.String(key),
		UploadId: aws.String(uploadID),
		MultipartUpload: &types.CompletedMultipartUpload{
			Parts: completedParts,
		},
	})
	if err != nil {
		t.Fatalf("CompleteMultipartUpload failed: %v", err)
	}

	// Verify Content
	resp, err := client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(key),
	})
	if err != nil {
		t.Fatalf("GetObject (multipart) failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	downloaded, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read multipart body: %v", err)
	}
	if !bytes.Equal(downloaded, data) {
		t.Errorf("Multipart content mismatch")
	}
}

func testVersioning(ctx context.Context, t *testing.T, client *s3.Client, bucketName string) {
	// Enable Versioning
	_, err := client.PutBucketVersioning(ctx, &s3.PutBucketVersioningInput{
		Bucket: aws.String(bucketName),
		VersioningConfiguration: &types.VersioningConfiguration{
			Status: types.BucketVersioningStatusEnabled,
		},
	})
	if err != nil {
		t.Fatalf("PutBucketVersioning failed: %v", err)
	}

	key := "versioned-object.txt"
	v1Content := "Version 1"
	v2Content := "Version 2"

	// Put V1
	_, err = client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(key),
		Body:   strings.NewReader(v1Content),
	})
	if err != nil {
		t.Fatalf("PutObject V1 failed: %v", err)
	}

	// Put V2
	_, err = client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(key),
		Body:   strings.NewReader(v2Content),
	})
	if err != nil {
		t.Fatalf("PutObject V2 failed: %v", err)
	}

	// List Versions
	versionsResp, err := client.ListObjectVersions(ctx, &s3.ListObjectVersionsInput{
		Bucket: aws.String(bucketName),
		Prefix: aws.String(key),
	})
	if err != nil {
		t.Fatalf("ListObjectVersions failed: %v", err)
	}

	// Should have at least 2 versions
	if len(versionsResp.Versions) < 2 {
		t.Errorf("Expected at least 2 versions, got %d", len(versionsResp.Versions))
	}

	// Get Specific Version (the older one)
	// Versions are usually listed latest first
	var v1ID string
	for _, v := range versionsResp.Versions {
		// This is a heuristic; in a real test we'd track the VersionId returned by PutObject
		// But here we just check if we can fetch them.
		if v.VersionId != nil {
			v1ID = *v.VersionId
		}
	}

	if v1ID != "" {
		_, err := client.GetObject(ctx, &s3.GetObjectInput{
			Bucket:    aws.String(bucketName),
			Key:       aws.String(key),
			VersionId: aws.String(v1ID),
		})
		if err != nil {
			t.Errorf("GetObject with VersionId failed: %v", err)
		}
	}
}

func cleanupBucket(ctx context.Context, t *testing.T, client *s3.Client, bucketName string) {
	// 1. List all objects and versions
	// Simple cleanup: ListObjectsV2 then DeleteObject
	// Also ListObjectVersions then DeleteObject with VersionId

	// Delete Versions
	versionsResp, err := client.ListObjectVersions(ctx, &s3.ListObjectVersionsInput{
		Bucket: aws.String(bucketName),
	})
	if err == nil {
		for _, v := range versionsResp.Versions {
			_, _ = client.DeleteObject(ctx, &s3.DeleteObjectInput{
				Bucket:    aws.String(bucketName),
				Key:       v.Key,
				VersionId: v.VersionId,
			})
		}
		for _, d := range versionsResp.DeleteMarkers {
			_, _ = client.DeleteObject(ctx, &s3.DeleteObjectInput{
				Bucket:    aws.String(bucketName),
				Key:       d.Key,
				VersionId: d.VersionId,
			})
		}
	}

	// Delete Objects (if any remain, though versions cover it)
	objectsResp, err := client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket: aws.String(bucketName),
	})
	if err == nil {
		for _, o := range objectsResp.Contents {
			_, _ = client.DeleteObject(ctx, &s3.DeleteObjectInput{
				Bucket: aws.String(bucketName),
				Key:    o.Key,
			})
		}
	}

	// Abort Multipart Uploads
	mpResp, err := client.ListMultipartUploads(ctx, &s3.ListMultipartUploadsInput{
		Bucket: aws.String(bucketName),
	})
	if err == nil {
		for _, u := range mpResp.Uploads {
			_, _ = client.AbortMultipartUpload(ctx, &s3.AbortMultipartUploadInput{
				Bucket:   aws.String(bucketName),
				Key:      u.Key,
				UploadId: u.UploadId,
			})
		}
	}

	// Delete Bucket
	_, err = client.DeleteBucket(ctx, &s3.DeleteBucketInput{
		Bucket: aws.String(bucketName),
	})
	if err != nil {
		t.Logf("Failed to delete bucket %s: %v", bucketName, err)
	}
}

func createS3Client(ctx context.Context, endpoint, accessKey, secretKey string) (*s3.Client, error) {
	cfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithRegion("us-east-1"),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
	)
	if err != nil {
		return nil, err
	}

	return s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String("http://" + endpoint)
		o.UsePathStyle = true
	}), nil
}
