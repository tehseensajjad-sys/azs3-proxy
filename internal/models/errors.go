package models

import (
	"net/http"
	"strings"
)

// S3ErrorCode represents standard S3 error codes that can be returned in API responses.
// These codes follow the S3 API specification and are used in error XML responses.
type S3ErrorCode string

const (
	// NoSuchKey is returned when a requested object/key does not exist
	NoSuchKey S3ErrorCode = "NoSuchKey"

	// NoSuchBucket is returned when a requested bucket does not exist
	NoSuchBucket S3ErrorCode = "NoSuchBucket"

	// BucketAlreadyExists is returned when attempting to create a bucket that already exists
	BucketAlreadyExists S3ErrorCode = "BucketAlreadyExists"

	// AccessDenied is returned when the request lacks proper authentication or permissions
	AccessDenied S3ErrorCode = "AccessDenied"

	// InvalidBucketName is returned when a bucket name does not meet S3 naming requirements
	InvalidBucketName S3ErrorCode = "InvalidBucketName"

	// InvalidArgument is returned when an argument is invalid
	InvalidArgument S3ErrorCode = "InvalidArgument"

	// InternalError is returned for unexpected server errors
	InternalError S3ErrorCode = "InternalError"

	// MalformedXML is returned when the XML provided was not well-formed
	MalformedXML S3ErrorCode = "MalformedXML"
)

// ErrorCodeToHTTPStatus maps S3 error codes to appropriate HTTP status codes.
// This ensures S3 error codes are properly translated to HTTP responses.
var ErrorCodeToHTTPStatus = map[S3ErrorCode]int{
	NoSuchKey:           http.StatusNotFound,            // 404
	NoSuchBucket:        http.StatusNotFound,            // 404
	BucketAlreadyExists: http.StatusConflict,            // 409
	AccessDenied:        http.StatusForbidden,           // 403
	InvalidBucketName:   http.StatusBadRequest,          // 400
	InvalidArgument:     http.StatusBadRequest,          // 400
	InternalError:       http.StatusInternalServerError, // 500
	MalformedXML:        http.StatusBadRequest,          // 400
}

// AzureErrorToS3 maps Azure Blob Storage error messages to S3 error codes.
// This allows Azure-specific errors to be converted to S3-compatible error responses.
// It performs case-insensitive matching against Azure error strings.
func AzureErrorToS3(azureError string) S3ErrorCode {
	// Convert error to lowercase for case-insensitive matching
	var errorStr = strings.ToLower(azureError)

	// Match Azure errors to appropriate S3 error codes
	switch {
	case strings.Contains(errorStr, "containernotfound"):
		// Azure container not found -> S3 bucket not found
		return NoSuchBucket

	case strings.Contains(errorStr, "blobnotfound"):
		// Azure blob not found -> S3 key not found
		return NoSuchKey

	case strings.Contains(errorStr, "containeralreadyexists"):
		// Azure container already exists -> S3 bucket already exists
		return BucketAlreadyExists

	case strings.Contains(errorStr, "authorizationpermissionmismatch"):
		// Azure authorization failure -> S3 access denied
		return AccessDenied

	case strings.Contains(errorStr, "invalidname"):
		// Azure invalid name -> S3 invalid bucket name
		return InvalidBucketName

	default:
		// Unknown Azure error -> return generic internal error
		return InternalError
	}
}

// S3Error represents an S3-compatible error with all details needed for an error response.
// It includes the error code, message, resource, and request ID.
type S3Error struct {
	Code      S3ErrorCode // S3 error code (e.g., NoSuchKey, AccessDenied)
	Message   string      // Detailed error message
	Resource  string      // The resource that caused the error (e.g., /bucket/key)
	RequestID string      // Unique request ID for tracking (optional)
}

// HTTPStatus returns the appropriate HTTP status code for this S3 error.
// Maps the S3 error code to the corresponding HTTP status, defaulting to 500 if unknown.
func (e *S3Error) HTTPStatus() int {
	if status, ok := ErrorCodeToHTTPStatus[e.Code]; ok {
		return status
	}
	return http.StatusInternalServerError
}

// Error implements the error interface for compatibility with Go's error interface.
// Returns the error message string.
func (e *S3Error) Error() string {
	return e.Message
}
