package models

import (
	"encoding/xml"
	"testing"
)

func TestListBucketsResponse(t *testing.T) {
	resp := &ListBucketsResponse{
		Buckets: []Bucket{
			{Name: "bucket1", CreationDate: "2024-01-01T00:00:00Z"},
		},
		Owner: Owner{ID: "owner-id", DisplayName: "Owner"},
	}

	data, err := xml.Marshal(resp)
	if err != nil {
		t.Errorf("xml.Marshal() failed: %v", err)
	}

	if len(data) == 0 {
		t.Error("xml.Marshal() returned empty data")
	}
}

func TestListObjectsResponse(t *testing.T) {
	resp := &ListObjectsResponse{
		Name:        "test-bucket",
		Prefix:      "test/",
		MaxKeys:     1000,
		IsTruncated: false,
		Contents: []Object{
			{Key: "test/file.txt", LastModified: "2024-01-01T00:00:00Z", ETag: "\"abc123\"", Size: 100, StorageClass: "STANDARD"},
		},
	}

	data, err := xml.Marshal(resp)
	if err != nil {
		t.Errorf("xml.Marshal() failed: %v", err)
	}

	if len(data) == 0 {
		t.Error("xml.Marshal() returned empty data")
	}
}

func TestListObjectsV2Response(t *testing.T) {
	resp := &ListObjectsV2Response{
		Name:                  "test-bucket",
		Prefix:                "test/",
		ContinuationToken:     "token-1",
		NextContinuationToken: "token-2",
		KeyCount:              2,
		MaxKeys:               2,
		IsTruncated:           true,
		Contents: []Object{
			{Key: "test/file1.txt", LastModified: "2024-01-01T00:00:00Z", ETag: "\"abc123\"", Size: 100, StorageClass: "STANDARD"},
			{Key: "test/file2.txt", LastModified: "2024-01-01T00:00:00Z", ETag: "\"def456\"", Size: 200, StorageClass: "STANDARD"},
		},
	}

	data, err := xml.Marshal(resp)
	if err != nil {
		t.Errorf("xml.Marshal() failed: %v", err)
	}

	if len(data) == 0 {
		t.Error("xml.Marshal() returned empty data")
	}
}

func TestErrorResponse(t *testing.T) {
	resp := &ErrorResponse{
		Code:      "NoSuchKey",
		Message:   "The specified key does not exist.",
		Resource:  "/bucket/key",
		RequestID: "request-123",
	}

	data, err := xml.Marshal(resp)
	if err != nil {
		t.Errorf("xml.Marshal() failed: %v", err)
	}

	if len(data) == 0 {
		t.Error("xml.Marshal() returned empty data")
	}
}
