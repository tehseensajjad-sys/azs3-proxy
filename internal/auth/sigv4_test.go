package auth

import (
	"testing"
	"time"
)

func TestVerifySignature(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		message string
		sig     string
		wantErr bool
	}{
		{name: "valid signature", key: "test-key", message: "test", sig: "valid", wantErr: false},
		{name: "invalid signature", key: "test-key", message: "test", sig: "invalid", wantErr: true},
		{name: "empty key", key: "", message: "test", sig: "sig", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := VerifySignature(tt.key, tt.message, tt.sig)
			if (err != nil) != tt.wantErr {
				t.Errorf("VerifySignature() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestBuildCanonicalRequest(t *testing.T) {
	tests := []struct {
		name   string
		method string
		path   string
		wantErr bool
	}{
		{name: "valid request", method: "GET", path: "/bucket/key", wantErr: false},
		{name: "empty method", method: "", path: "/bucket/key", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := BuildCanonicalRequest(tt.method, tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("BuildCanonicalRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
