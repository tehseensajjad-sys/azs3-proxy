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

func TestS3ErrorError(t *testing.T) {
	tests := []struct {
		name        string
		err         *S3Error
		shouldError bool
	}{
		{
			name: "error with code and message",
			err: &S3Error{
				Code:    NoSuchKey,
				Message: "Key not found",
			},
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err != nil {
				errStr := tt.err.Error()
				if tt.shouldError && errStr == "" {
					t.Error("Expected non-empty error string")
				}
			}
		})
	}
}

func TestAzureErrorToS3_AllMappings(t *testing.T) {
	mappings := map[string]S3ErrorCode{
		"ContainerNotFound":               NoSuchBucket,
		"BlobNotFound":                    NoSuchKey,
		"ContainerAlreadyExists":          BucketAlreadyExists,
		"InvalidName":                     InvalidBucketName,
		"AuthorizationPermissionMismatch": AccessDenied,
		"UnknownError":                    InternalError,
	}

	for azureErr, expectedCode := range mappings {
		t.Run(azureErr, func(t *testing.T) {
			code := AzureErrorToS3(azureErr)
			if code != expectedCode {
				t.Errorf("AzureErrorToS3(%q) = %v, want %v", azureErr, code, expectedCode)
			}
		})
	}
}

func TestHTTPStatus_AllErrorCodes(t *testing.T) {
	tests := []struct {
		code           S3ErrorCode
		expectedStatus int
	}{
		{NoSuchBucket, 404},
		{NoSuchKey, 404},
		{BucketAlreadyExists, 409},
		{AccessDenied, 403},
		{InvalidBucketName, 400},
		{InternalError, 500},
	}

	for _, tt := range tests {
		t.Run(string(tt.code), func(t *testing.T) {
			err := &S3Error{Code: tt.code}
			status := err.HTTPStatus()
			if status != tt.expectedStatus {
				t.Errorf("HTTPStatus() for %v = %d, want %d", tt.code, status, tt.expectedStatus)
			}
		})
	}
}

// TestHTTPStatus_UnknownErrorCode tests default status for unknown error codes
func TestHTTPStatus_UnknownErrorCode(t *testing.T) {
	err := &S3Error{Code: S3ErrorCode("UnknownCode")}
	status := err.HTTPStatus()
	if status != 500 {
		t.Errorf("HTTPStatus() for unknown code = %d, want 500", status)
	}
}
