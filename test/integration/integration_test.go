//go:build integration

package integration

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/vibhansa-msft/azs3-proxy/internal/config"
	"github.com/vibhansa-msft/azs3-proxy/internal/server"
)

// TestIntegration runs a full integration test suite against the proxy and a real/emulated backend.
// It requires AZURE_STORAGE_CONNECTION_STRING or AZURE_STORAGE_ACCOUNT/KEY to be set.
// If not set, it skips the test.
func TestIntegration(t *testing.T) {
	// 1. Check environment requirements
	connStr := os.Getenv("AZURE_STORAGE_CONNECTION_STRING")
	accountName := os.Getenv("AZURE_STORAGE_ACCOUNT")
	accountKey := os.Getenv("AZURE_STORAGE_KEY")

	if connStr == "" && (accountName == "" || accountKey == "") {
		t.Skip("Skipping integration test: AZURE_STORAGE_CONNECTION_STRING or AZURE_STORAGE_ACCOUNT/KEY not set")
	}

	// 2. Setup Proxy Server
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Find a free port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to listen on random port: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	_ = listener.Close() // Close it, we just wanted the port. The server will listen on it.

	proxyAddr := fmt.Sprintf("127.0.0.1:%d", port)

	// Configure Proxy
	cfg := &config.Config{
		ListenAddr:        proxyAddr,
		S3AccessKeyID:     "test-access-key",
		S3SecretAccessKey: "test-secret-key",
		LogLevel:          "error", // Keep logs quiet during test
	}

	if connStr != "" {
		// Parse connection string to get account info (simplified for test)
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
		storageURL := fmt.Sprintf("https://%s.blob.core.windows.net", accountName)
		if accountName == "devstoreaccount1" {
			storageURL = "http://127.0.0.1:10000/devstoreaccount1"
		}
		cfg.AzureAuth = &config.AzureAuthConfig{
			Mode:               config.AuthModeAccountKey,
			StorageAccountName: accountName,
			AccountKey:         accountKey,
			StorageAccountURL:  storageURL,
		}
	}

	logger, _ := zap.NewDevelopment()
	router := chi.NewRouter()

	// Initialize Server
	srv, err := server.NewS3ProxyServer(router, cfg, logger, nil)
	if err != nil {
		t.Fatalf("Failed to create proxy server: %v", err)
	}
	defer func() { _ = srv.Close() }()

	// Start Server in Goroutine
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

	// Wait for server to start
	time.Sleep(100 * time.Millisecond)

	// 3. Setup S3 Client
	s3Client, err := createS3Client(ctx, proxyAddr, cfg.S3AccessKeyID, cfg.S3SecretAccessKey)
	if err != nil {
		t.Fatalf("Failed to create S3 client: %v", err)
	}

	// 4. Run Tests
	bucketName := fmt.Sprintf("test-bucket-%d", time.Now().UnixNano())
	objectKey := "test-object.txt"
	objectContent := "Hello, S3 Proxy!"

	t.Run("CreateBucket", func(t *testing.T) {
		_, err := s3Client.CreateBucket(ctx, &s3.CreateBucketInput{
			Bucket: aws.String(bucketName),
		})
		if err != nil {
			t.Fatalf("CreateBucket failed: %v", err)
		}
	})

	t.Run("ListBuckets", func(t *testing.T) {
		resp, err := s3Client.ListBuckets(ctx, &s3.ListBucketsInput{})
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
	})

	t.Run("PutObject", func(t *testing.T) {
		_, err := s3Client.PutObject(ctx, &s3.PutObjectInput{
			Bucket: aws.String(bucketName),
			Key:    aws.String(objectKey),
			Body:   strings.NewReader(objectContent),
		})
		if err != nil {
			t.Fatalf("PutObject failed: %v", err)
		}
	})

	t.Run("GetObject", func(t *testing.T) {
		resp, err := s3Client.GetObject(ctx, &s3.GetObjectInput{
			Bucket: aws.String(bucketName),
			Key:    aws.String(objectKey),
		})
		if err != nil {
			t.Fatalf("GetObject failed: %v", err)
		}
		defer func() { _ = resp.Body.Close() }()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("Failed to read object body: %v", err)
		}

		if string(body) != objectContent {
			t.Errorf("Content mismatch. Got %q, want %q", string(body), objectContent)
		}
	})

	t.Run("ListObjectsV2", func(t *testing.T) {
		resp, err := s3Client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
			Bucket: aws.String(bucketName),
		})
		if err != nil {
			t.Fatalf("ListObjectsV2 failed: %v", err)
		}

		found := false
		for _, obj := range resp.Contents {
			if *obj.Key == objectKey {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Object %s not found in list", objectKey)
		}
	})

	t.Run("DeleteObject", func(t *testing.T) {
		_, err := s3Client.DeleteObject(ctx, &s3.DeleteObjectInput{
			Bucket: aws.String(bucketName),
			Key:    aws.String(objectKey),
		})
		if err != nil {
			t.Fatalf("DeleteObject failed: %v", err)
		}
	})

	t.Run("DeleteBucket", func(t *testing.T) {
		_, err := s3Client.DeleteBucket(ctx, &s3.DeleteBucketInput{
			Bucket: aws.String(bucketName),
		})
		if err != nil {
			t.Fatalf("DeleteBucket failed: %v", err)
		}
	})
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
		o.UsePathStyle = true // Required for local testing/proxies usually
	}), nil
}
