package models

import (
	"net/http"
	"strings"
)

// S3ErrorCode represents S3 error codes
type S3ErrorCode string

const (
	NoSuchKey           S3ErrorCode = "NoSuchKey"
	NoSuchBucket        S3ErrorCode = "NoSuchBucket"
	BucketAlreadyExists S3ErrorCode = "BucketAlreadyExists"
	AccessDenied        S3ErrorCode = "AccessDenied"
	InvalidBucketName   S3ErrorCode = "InvalidBucketName"
	InternalError       S3ErrorCode = "InternalError"
)

// ErrorCodeToHTTPStatus maps S3 error codes to HTTP status codes
var ErrorCodeToHTTPStatus = map[S3ErrorCode]int{
	NoSuchKey:           http.StatusNotFound,
	NoSuchBucket:        http.StatusNotFound,
	BucketAlreadyExists: http.StatusConflict,
	AccessDenied:        http.StatusForbidden,
	InvalidBucketName:   http.StatusBadRequest,
	InternalError:       http.StatusInternalServerError,
}

// AzureErrorToS3 maps Azure error codes to S3 error codes
func AzureErrorToS3(azureError string) S3ErrorCode {
	var errorStr = strings.ToLower(azureError)

	switch {
	case strings.Contains(errorStr, "containernotfound"):
		return NoSuchBucket
	case strings.Contains(errorStr, "blobnotfound"):
		return NoSuchKey
	case strings.Contains(errorStr, "containeralreadyexists"):
		return BucketAlreadyExists
	case strings.Contains(errorStr, "authorizationpermissionmismatch"):
		return AccessDenied
	case strings.Contains(errorStr, "invalidname"):
		return InvalidBucketName
	default:
		return InternalError
	}
}

// S3Error represents an S3-compatible error response
type S3Error struct {
	Code      S3ErrorCode
	Message   string
	Resource  string
	RequestID string
}

// HTTPStatus returns the HTTP status code for the error
func (e *S3Error) HTTPStatus() int {
	if status, ok := ErrorCodeToHTTPStatus[e.Code]; ok {
		return status
	}
	return http.StatusInternalServerError
}

// Error implements the error interface
func (e *S3Error) Error() string {
	return e.Message
}
