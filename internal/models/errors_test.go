package models

import (
	"testing"
)

func TestAzureErrorToS3(t *testing.T) {
	tests := []struct {
		name     string
		azureErr string
		expected S3ErrorCode
	}{
		{name: "container not found", azureErr: "ContainerNotFound", expected: NoSuchBucket},
		{name: "blob not found", azureErr: "BlobNotFound", expected: NoSuchKey},
		{name: "container already exists", azureErr: "ContainerAlreadyExists", expected: BucketAlreadyExists},
		{name: "authorization error", azureErr: "AuthorizationPermissionMismatch", expected: AccessDenied},
		{name: "invalid name", azureErr: "InvalidName", expected: InvalidBucketName},
		{name: "unknown error", azureErr: "UnknownError", expected: InternalError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code := AzureErrorToS3(tt.azureErr)
			if code != tt.expected {
				t.Errorf("AzureErrorToS3() got %v, want %v", code, tt.expected)
			}
		})
	}
}

func TestS3ErrorHTTPStatus(t *testing.T) {
	tests := []struct {
		name     string
		err      *S3Error
		expected int
	}{
		{name: "no such key", err: &S3Error{Code: NoSuchKey}, expected: 404},
		{name: "access denied", err: &S3Error{Code: AccessDenied}, expected: 403},
		{name: "bucket already exists", err: &S3Error{Code: BucketAlreadyExists}, expected: 409},
		{name: "internal error", err: &S3Error{Code: InternalError}, expected: 500},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code := tt.err.HTTPStatus()
			if code != tt.expected {
				t.Errorf("HTTPStatus() got %v, want %v", code, tt.expected)
			}
		})
	}
}
