package models

import (
	"encoding/xml"
)

// ListBucketsResponse represents the response for list buckets operation
type ListBucketsResponse struct {
	XMLName xml.Name  `xml:"ListAllMyBucketsResult"`
	Buckets []Bucket  `xml:"Buckets>Bucket"`
	Owner   Owner     `xml:"Owner"`
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
	XMLName xml.Name `xml:"InitiateMultipartUploadResult"`
	Bucket  string   `xml:"Bucket"`
	Key     string   `xml:"Key"`
	UploadID string   `xml:"UploadId"`
}

// CompleteMultipartUploadResponse represents the response for complete multipart upload
type CompleteMultipartUploadResponse struct {
	XMLName xml.Name `xml:"CompleteMultipartUploadResult"`
	Bucket  string   `xml:"Bucket"`
	Key     string   `xml:"Key"`
	ETag    string   `xml:"ETag"`
}

// ErrorResponse represents an S3 error response
type ErrorResponse struct {
	XMLName   xml.Name `xml:"Error"`
	Code      string   `xml:"Code"`
	Message   string   `xml:"Message"`
	Resource  string   `xml:"Resource,omitempty"`
	RequestID string   `xml:"RequestId,omitempty"`
}
