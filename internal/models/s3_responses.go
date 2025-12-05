package models

import (
	"encoding/xml"
)

// ListBucketsResponse represents the response for list buckets operation
type ListBucketsResponse struct {
	XMLName xml.Name `xml:"ListAllMyBucketsResult"`
	Buckets []Bucket `xml:"Buckets>Bucket"`
	Owner   Owner    `xml:"Owner"`
}

// Bucket represents a single bucket in the list
type Bucket struct {
	Name         string `xml:"Name"`
	CreationDate string `xml:"CreationDate"`
}

// Owner represents the owner of a bucket
type Owner struct {
	ID          string `xml:"ID"`
	DisplayName string `xml:"DisplayName"`
}

// ListObjectsResponse represents the response for list objects operation
type ListObjectsResponse struct {
	XMLName        xml.Name `xml:"ListBucketResult"`
	Name           string   `xml:"Name"`
	Prefix         string   `xml:"Prefix"`
	Marker         string   `xml:"Marker"`
	MaxKeys        int      `xml:"MaxKeys"`
	IsTruncated    bool     `xml:"IsTruncated"`
	Contents       []Object `xml:"Contents"`
	CommonPrefixes []string `xml:"CommonPrefixes>Prefix"`
}

// Object represents a single object in the list
type Object struct {
	Key          string `xml:"Key"`
	LastModified string `xml:"LastModified"`
	ETag         string `xml:"ETag"`
	Size         int64  `xml:"Size"`
	StorageClass string `xml:"StorageClass"`
}

// CopyObjectResponse represents the response for copy object operation
type CopyObjectResponse struct {
	XMLName      xml.Name `xml:"CopyObjectResult"`
	LastModified string   `xml:"LastModified"`
	ETag         string   `xml:"ETag"`
}

// InitiateMultipartUploadResponse represents the response for initiate multipart upload
type InitiateMultipartUploadResponse struct {
	XMLName  xml.Name `xml:"InitiateMultipartUploadResult"`
	Bucket   string   `xml:"Bucket"`
	Key      string   `xml:"Key"`
	UploadID string   `xml:"UploadId"`
}

// CompleteMultipartUploadResponse represents the response for complete multipart upload
type CompleteMultipartUploadResponse struct {
	XMLName xml.Name `xml:"CompleteMultipartUploadResult"`
	Bucket  string   `xml:"Bucket"`
	Key     string   `xml:"Key"`
	ETag    string   `xml:"ETag"`
}

// Part represents a single part in a multipart upload
type Part struct {
	PartNumber int    `xml:"PartNumber"`
	ETag       string `xml:"ETag"`
	Size       int64  `xml:"Size"`
}

// ListPartsResponse represents the response for listing parts of a multipart upload
type ListPartsResponse struct {
	XMLName              xml.Name `xml:"ListPartsResult"`
	Bucket               string   `xml:"Bucket"`
	Key                  string   `xml:"Key"`
	UploadID             string   `xml:"UploadId"`
	StorageClass         string   `xml:"StorageClass"`
	PartNumberMarker     int      `xml:"PartNumberMarker"`
	NextPartNumberMarker int      `xml:"NextPartNumberMarker"`
	MaxParts             int      `xml:"MaxParts"`
	IsTruncated          bool     `xml:"IsTruncated"`
	Parts                []Part   `xml:"Parts>Part"`
}

// ListMultipartUploadsResponse represents the response for listing ongoing multipart uploads
type ListMultipartUploadsResponse struct {
	XMLName            xml.Name     `xml:"ListMultipartUploadResult"`
	Bucket             string       `xml:"Bucket"`
	KeyMarker          string       `xml:"KeyMarker"`
	UploadIDMarker     string       `xml:"UploadIdMarker"`
	NextKeyMarker      string       `xml:"NextKeyMarker"`
	NextUploadIDMarker string       `xml:"NextUploadIdMarker"`
	MaxUploads         int          `xml:"MaxUploads"`
	IsTruncated        bool         `xml:"IsTruncated"`
	Uploads            []UploadInfo `xml:"Uploads>Upload"`
}

// UploadInfo represents a single ongoing multipart upload
type UploadInfo struct {
	Key          string `xml:"Key"`
	UploadID     string `xml:"UploadId"`
	Initiated    string `xml:"Initiated"`
	StorageClass string `xml:"StorageClass"`
}

// VersioningConfiguration represents bucket versioning status
type VersioningConfiguration struct {
	XMLName xml.Name `xml:"VersioningConfiguration"`
	Status  string   `xml:"Status"` // Enabled, Suspended, or empty
}

// ListObjectVersionsResponse represents the response for list object versions
type ListObjectVersionsResponse struct {
	XMLName             xml.Name           `xml:"ListVersionsResult"`
	Name                string             `xml:"Name"`
	Prefix              string             `xml:"Prefix"`
	KeyMarker           string             `xml:"KeyMarker"`
	VersionIDMarker     string             `xml:"VersionIdMarker"`
	NextKeyMarker       string             `xml:"NextKeyMarker"`
	NextVersionIDMarker string             `xml:"NextVersionIdMarker"`
	MaxKeys             int                `xml:"MaxKeys"`
	IsTruncated         bool               `xml:"IsTruncated"`
	Versions            []ObjectVersionXML `xml:"Versions>Version"`
	DeleteMarkers       []DeleteMarkerXML  `xml:"DeleteMarker"`
}

// ObjectVersionXML represents a single object version in XML response
type ObjectVersionXML struct {
	Key          string `xml:"Key"`
	VersionID    string `xml:"VersionId"`
	IsLatest     bool   `xml:"IsLatest"`
	LastModified string `xml:"LastModified"`
	ETag         string `xml:"ETag"`
	Size         int64  `xml:"Size"`
	StorageClass string `xml:"StorageClass"`
}

// DeleteMarkerXML represents a delete marker in version list
type DeleteMarkerXML struct {
	Key          string `xml:"Key"`
	VersionID    string `xml:"VersionId"`
	IsLatest     bool   `xml:"IsLatest"`
	LastModified string `xml:"LastModified"`
	Owner        Owner  `xml:"Owner"`
}

// ErrorResponse represents an S3 error response
type ErrorResponse struct {
	XMLName   xml.Name `xml:"Error"`
	Code      string   `xml:"Code"`
	Message   string   `xml:"Message"`
	Resource  string   `xml:"Resource,omitempty"`
	RequestID string   `xml:"RequestId,omitempty"`
}
