package models

import (
	"encoding/xml"
)

// ListBucketsResponse represents the S3 API response for the ListBuckets operation.
// It contains all buckets owned by the account and owner information.
type ListBucketsResponse struct {
	XMLName xml.Name `xml:"ListAllMyBucketsResult"` // XML root element name
	Buckets []Bucket `xml:"Buckets>Bucket"`         // List of buckets
	Owner   Owner    `xml:"Owner"`                  // Owner of the buckets
}

// Bucket represents a single bucket in the list buckets response.
// Contains the bucket name and creation date.
type Bucket struct {
	Name         string `xml:"Name"`         // Name of the bucket
	CreationDate string `xml:"CreationDate"` // ISO 8601 formatted creation timestamp
}

// Owner represents the owner of buckets in the account.
// Contains owner identification and display information.
type Owner struct {
	ID          string `xml:"ID"`          // Owner's unique identifier
	DisplayName string `xml:"DisplayName"` // Owner's display name (optional in response)
}

// ListObjectsResponse represents the S3 API response for the legacy ListObjects operation.
// It contains objects in a bucket, pagination info, and optionally common prefixes for delimiter-based listing.
type ListObjectsResponse struct {
	XMLName        xml.Name `xml:"ListBucketResult"`      // XML root element name
	Name           string   `xml:"Name"`                  // Bucket name
	Prefix         string   `xml:"Prefix"`                // Request prefix filter
	Marker         string   `xml:"Marker"`                // Pagination marker for next results
	MaxKeys        int      `xml:"MaxKeys"`               // Maximum keys returned
	IsTruncated    bool     `xml:"IsTruncated"`           // Whether more results exist
	Contents       []Object `xml:"Contents"`              // List of objects
	CommonPrefixes []string `xml:"CommonPrefixes>Prefix"` // Common prefixes when using delimiter
}

// ListObjectsV2Response represents the S3 API response for the ListObjectsV2 operation.
// It includes V2-specific pagination tokens and key counts.
type ListObjectsV2Response struct {
	XMLName               xml.Name `xml:"ListBucketResult"` // XML root element name
	Name                  string   `xml:"Name"`             // Bucket name
	Prefix                string   `xml:"Prefix"`           // Request prefix filter
	ContinuationToken     string   `xml:"ContinuationToken,omitempty"`
	NextContinuationToken string   `xml:"NextContinuationToken,omitempty"`
	KeyCount              int      `xml:"KeyCount"`              // Number of keys in this response
	MaxKeys               int      `xml:"MaxKeys"`               // Maximum keys returned
	IsTruncated           bool     `xml:"IsTruncated"`           // Whether more results exist
	Contents              []Object `xml:"Contents"`              // List of objects
	CommonPrefixes        []string `xml:"CommonPrefixes>Prefix"` // Common prefixes when using delimiter
}

// Object represents a single object/key in the list objects response.
// Contains object metadata like name, size, modification time, and storage class.
type Object struct {
	Key          string `xml:"Key"`          // Object key/name
	LastModified string `xml:"LastModified"` // ISO 8601 formatted last modification timestamp
	ETag         string `xml:"ETag"`         // Object entity tag (usually MD5 hash)
	Size         int64  `xml:"Size"`         // Object size in bytes
	StorageClass string `xml:"StorageClass"` // Storage class (e.g., STANDARD)
}

// CopyObjectResult represents the response for a successful object copy operation.
type CopyObjectResult struct {
	XMLName      xml.Name `xml:"CopyObjectResult"`
	LastModified string   `xml:"LastModified"`
	ETag         string   `xml:"ETag"`
}

// CopyObjectResponse represents the S3 API response for the CopyObject operation.
// Contains metadata of the newly copied object.
type CopyObjectResponse struct {
	XMLName      xml.Name `xml:"CopyObjectResult"` // XML root element name
	LastModified string   `xml:"LastModified"`     // ISO 8601 formatted modification timestamp
	ETag         string   `xml:"ETag"`             // Entity tag of the copied object
}

// InitiateMultipartUploadResponse represents the S3 API response for the InitiateMultipartUpload operation.
// Returns the upload ID needed for subsequent part uploads and completion.
type InitiateMultipartUploadResponse struct {
	XMLName  xml.Name `xml:"InitiateMultipartUploadResult"` // XML root element name
	Bucket   string   `xml:"Bucket"`                        // Bucket name
	Key      string   `xml:"Key"`                           // Object key
	UploadID string   `xml:"UploadId"`                      // Unique upload ID for this multipart upload
}

// CompleteMultipartUploadResponse represents the S3 API response for the CompleteMultipartUpload operation.
// Returns final object information after all parts are uploaded and combined.
type CompleteMultipartUploadResponse struct {
	XMLName xml.Name `xml:"CompleteMultipartUploadResult"` // XML root element name
	Bucket  string   `xml:"Bucket"`                        // Bucket name
	Key     string   `xml:"Key"`                           // Object key
	ETag    string   `xml:"ETag"`                          // Final object ETag
}

// Part represents a single part in a multipart upload.
// Contains part number, ETag, and size information.
type Part struct {
	PartNumber int    `xml:"PartNumber"` // Part number (1-10000)
	ETag       string `xml:"ETag"`       // Part entity tag
	Size       int64  `xml:"Size"`       // Part size in bytes
}

// ListPartsResponse represents the S3 API response for the ListParts operation.
// Lists all parts that have been uploaded for a specific multipart upload.
type ListPartsResponse struct {
	XMLName              xml.Name `xml:"ListPartsResult"`      // XML root element name
	Bucket               string   `xml:"Bucket"`               // Bucket name
	Key                  string   `xml:"Key"`                  // Object key
	UploadID             string   `xml:"UploadId"`             // Upload ID
	StorageClass         string   `xml:"StorageClass"`         // Storage class
	PartNumberMarker     int      `xml:"PartNumberMarker"`     // Pagination marker for parts
	NextPartNumberMarker int      `xml:"NextPartNumberMarker"` // Next pagination marker
	MaxParts             int      `xml:"MaxParts"`             // Maximum parts to return
	IsTruncated          bool     `xml:"IsTruncated"`          // Whether more parts exist
	Parts                []Part   `xml:"Parts>Part"`           // List of uploaded parts
}

// ListMultipartUploadsResponse represents the S3 API response for the ListMultipartUploads operation.
// Lists all ongoing multipart uploads in a bucket.
type ListMultipartUploadsResponse struct {
	XMLName            xml.Name     `xml:"ListMultipartUploadResult"` // XML root element name
	Bucket             string       `xml:"Bucket"`                    // Bucket name
	KeyMarker          string       `xml:"KeyMarker"`                 // Pagination marker for keys
	UploadIDMarker     string       `xml:"UploadIdMarker"`            // Pagination marker for upload IDs
	NextKeyMarker      string       `xml:"NextKeyMarker"`             // Next key marker
	NextUploadIDMarker string       `xml:"NextUploadIdMarker"`        // Next upload ID marker
	MaxUploads         int          `xml:"MaxUploads"`                // Maximum uploads to return
	IsTruncated        bool         `xml:"IsTruncated"`               // Whether more uploads exist
	Uploads            []UploadInfo `xml:"Uploads>Upload"`            // List of ongoing uploads
}

// UploadInfo represents a single ongoing multipart upload.
// Contains upload identification and status information.
type UploadInfo struct {
	Key          string `xml:"Key"`          // Object key
	UploadID     string `xml:"UploadId"`     // Unique upload ID
	Initiated    string `xml:"Initiated"`    // ISO 8601 formatted initiation timestamp
	StorageClass string `xml:"StorageClass"` // Storage class
}

// VersioningConfiguration represents the bucket versioning status.
// Can be Enabled, Suspended, or empty (not set).
type VersioningConfiguration struct {
	XMLName xml.Name `xml:"VersioningConfiguration"` // XML root element name
	Status  string   `xml:"Status"`                  // Status: "Enabled", "Suspended", or empty
}

// ListObjectVersionsResponse represents the S3 API response for the ListObjectVersions operation.
// Lists all versions of objects in a bucket, including delete markers.
type ListObjectVersionsResponse struct {
	XMLName             xml.Name           `xml:"ListVersionsResult"`  // XML root element name
	Name                string             `xml:"Name"`                // Bucket name
	Prefix              string             `xml:"Prefix"`              // Request prefix filter
	KeyMarker           string             `xml:"KeyMarker"`           // Pagination marker for keys
	VersionIDMarker     string             `xml:"VersionIdMarker"`     // Pagination marker for version IDs
	NextKeyMarker       string             `xml:"NextKeyMarker"`       // Next key marker
	NextVersionIDMarker string             `xml:"NextVersionIdMarker"` // Next version ID marker
	MaxKeys             int                `xml:"MaxKeys"`             // Maximum keys to return
	IsTruncated         bool               `xml:"IsTruncated"`         // Whether more versions exist
	Versions            []ObjectVersionXML `xml:"Versions>Version"`    // List of object versions
	DeleteMarkers       []DeleteMarkerXML  `xml:"DeleteMarker"`        // List of delete markers
}

// ObjectVersionXML represents a single object version in the list versions response.
// Contains version metadata and indicates if it's the latest version.
type ObjectVersionXML struct {
	Key          string `xml:"Key"`          // Object key
	VersionID    string `xml:"VersionId"`    // Unique version ID
	IsLatest     bool   `xml:"IsLatest"`     // Whether this is the latest version
	LastModified string `xml:"LastModified"` // ISO 8601 formatted modification timestamp
	ETag         string `xml:"ETag"`         // Version entity tag
	Size         int64  `xml:"Size"`         // Object size in bytes
	StorageClass string `xml:"StorageClass"` // Storage class
}

// DeleteMarkerXML represents a delete marker in the list versions response.
// A delete marker is created when an object is deleted with versioning enabled.
type DeleteMarkerXML struct {
	Key          string `xml:"Key"`          // Object key
	VersionID    string `xml:"VersionId"`    // Unique version ID of the delete marker
	IsLatest     bool   `xml:"IsLatest"`     // Whether this is the latest delete marker
	LastModified string `xml:"LastModified"` // ISO 8601 formatted deletion timestamp
	Owner        Owner  `xml:"Owner"`        // Owner who deleted the object
}

// ErrorResponse represents an S3 API error response in XML format.
// Contains error details to be returned when an operation fails.
type ErrorResponse struct {
	XMLName   xml.Name `xml:"Error"`               // XML root element name
	Code      string   `xml:"Code"`                // S3 error code
	Message   string   `xml:"Message"`             // Human-readable error message
	Resource  string   `xml:"Resource,omitempty"`  // Resource that caused the error (optional)
	RequestID string   `xml:"RequestId,omitempty"` // Request ID for tracking (optional)
}

// DeleteObjectsRequest represents the S3 API request for the DeleteObjects operation.
type DeleteObjectsRequest struct {
	XMLName xml.Name           `xml:"Delete"`
	Quiet   bool               `xml:"Quiet"`
	Objects []ObjectIdentifier `xml:"Object"`
}

// ObjectIdentifier represents an object to delete in DeleteObjectsRequest.
type ObjectIdentifier struct {
	Key       string `xml:"Key"`
	VersionID string `xml:"VersionId,omitempty"`
}

// DeleteResult represents the S3 API response for the DeleteObjects operation.
type DeleteResult struct {
	XMLName xml.Name        `xml:"DeleteResult"`
	Deleted []DeletedObject `xml:"Deleted"`
	Error   []ErrorResult   `xml:"Error"`
}

// DeletedObject represents a successfully deleted object in DeleteResult.
type DeletedObject struct {
	Key                   string `xml:"Key"`
	VersionID             string `xml:"VersionId,omitempty"`
	DeleteMarker          bool   `xml:"DeleteMarker,omitempty"`
	DeleteMarkerVersionID string `xml:"DeleteMarkerVersionId,omitempty"`
}

// ErrorResult represents a failed deletion in DeleteResult.
type ErrorResult struct {
	Key       string `xml:"Key"`
	VersionID string `xml:"VersionId,omitempty"`
	Code      string `xml:"Code"`
	Message   string `xml:"Message"`
}
