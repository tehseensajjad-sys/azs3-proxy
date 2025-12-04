package models

import (
	"encoding/xml"
	"testing"
)

func TestListBucketsResponse(t *testing.T) {
	tests := []struct {
		name     string
		buckets  []string
		wantErr  bool
	}{
		{name: "single bucket", buckets: []string{"test-bucket"}, wantErr: false},
		{name: "multiple buckets", buckets: []string{"bucket1", "bucket2"}, wantErr: false},
		{name: "empty buckets", buckets: []string{}, wantErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := NewListBucketsResponse(tt.buckets)
			if resp == nil {
				t.Error("NewListBucketsResponse() returned nil")
			}
		})
	}
}

func TestMarshalXML(t *testing.T) {
	tests := []struct {
		name    string
		obj     interface{}
		wantErr bool
	}{
		{name: "list buckets response", obj: &ListBucketsResponse{}, wantErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := xml.Marshal(tt.obj)
			if (err != nil) != tt.wantErr {
				t.Errorf("xml.Marshal() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestListObjectsResponse(t *testing.T) {
	tests := []struct {
		name      string
		bucket    string
		objects   int
		wantErr   bool
	}{
		{name: "single object", bucket: "test-bucket", objects: 1, wantErr: false},
		{name: "multiple objects", bucket: "test-bucket", objects: 100, wantErr: false},
		{name: "no objects", bucket: "test-bucket", objects: 0, wantErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := NewListObjectsResponse(tt.bucket, tt.objects)
			if resp == nil {
				t.Error("NewListObjectsResponse() returned nil")
			}
		})
	}
}
