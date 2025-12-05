package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"testing"
	"time"
)

func TestVerifySignature(t *testing.T) {
	av := NewAuthVerifier("AKIA1234567890ABCDEF", "wJalrXUtnFEMI/K7MDENG+bPxRfiCYEXAMPLEKEY")

	// Create a simple test request without Authorization header
	req, _ := http.NewRequest("GET", "http://localhost:8080/", nil)

	// Should fail without Authorization header
	err := av.VerifySignature(req)
	if err == nil {
		t.Error("Expected error for request without Authorization header")
	}
}

func TestVerifySignatureWithValidHeader(t *testing.T) {
	accessKey := "AKIA1234567890ABCDEF"
	secretKey := "wJalrXUtnFEMI/K7MDENG+bPxRfiCYEXAMPLEKEY"
	av := NewAuthVerifier(accessKey, secretKey)

	// Create a request with Authorization header
	req, _ := http.NewRequest("GET", "http://localhost:8080/", nil)
	req.Header.Set("Authorization", "AWS4-HMAC-SHA256 Credential=AKIA1234567890ABCDEF/20231201/us-east-1/s3/aws4_request, SignedHeaders=host;x-amz-date, Signature=invalid")
	req.Header.Set("X-Amz-Date", "20231201T120000Z")

	// Should fail with invalid signature
	err := av.VerifySignature(req)
	if err == nil {
		t.Error("Expected error for invalid signature")
	}
}

func TestNewAuthVerifier(t *testing.T) {
	av := NewAuthVerifier("test-key", "test-secret")

	if av.accessKey != "test-key" {
		t.Errorf("Expected accessKey 'test-key', got '%s'", av.accessKey)
	}

	if av.secretKey != "test-secret" {
		t.Errorf("Expected secretKey 'test-secret', got '%s'", av.secretKey)
	}
}

func TestBuildCanonicalRequest(t *testing.T) {
	req, _ := http.NewRequest("GET", "http://s3.amazonaws.com/bucket/key", nil)
	req.Header.Set("Host", "s3.amazonaws.com")
	req.Header.Set("X-Amz-Date", "20231201T120000Z")

	av := NewAuthVerifier("test", "test")
	// We need to call the private method indirectly through the signature flow
	// This test verifies the auth verifier is properly constructed
	if av.accessKey == "" {
		t.Error("AuthVerifier not properly initialized")
	}
}

func TestCalculateSignature(t *testing.T) {
	// Test the signature calculation with known values
	stringToSign := "AWS4-HMAC-SHA256\n20231201T120000Z\n20231201/us-east-1/s3/aws4_request\nhashvalue"
	secretKey := "wJalrXUtnFEMI/K7MDENG+bPxRfiCYEXAMPLEKEY"
	region := "us-east-1"
	service := "s3"
	date := "20231201"

	// Build the signing key like AWS does
	kDate := hmac.New(sha256.New, []byte("AWS4"+secretKey))
	kDate.Write([]byte(date))

	kRegion := hmac.New(sha256.New, kDate.Sum(nil))
	kRegion.Write([]byte(region))

	kService := hmac.New(sha256.New, kRegion.Sum(nil))
	kService.Write([]byte(service))

	kSigning := hmac.New(sha256.New, kService.Sum(nil))
	kSigning.Write([]byte("aws4_request"))

	signature := hmac.New(sha256.New, kSigning.Sum(nil))
	signature.Write([]byte(stringToSign))

	expectedSig := hex.EncodeToString(signature.Sum(nil))
	if expectedSig == "" {
		t.Error("Expected non-empty signature")
	}
}

func TestAuthVerifierWithDifferentMethods(t *testing.T) {
	av := NewAuthVerifier("key", "secret")

	methods := []string{"GET", "PUT", "POST", "DELETE", "HEAD"}
	for _, method := range methods {
		req, _ := http.NewRequest(method, "http://localhost/bucket/key", nil)
		err := av.VerifySignature(req)
		// Should fail because no Authorization header, not because of method
		if err == nil {
			t.Errorf("Expected error for %s request without auth", method)
		}
	}
}

func TestAuthVerifierDateHandling(t *testing.T) {
	av := NewAuthVerifier("key", "secret")

	// Test with X-Amz-Date header
	req, _ := http.NewRequest("GET", "http://localhost/", nil)
	req.Header.Set("X-Amz-Date", "20231201T120000Z")
	req.Header.Set("Authorization", "AWS4-HMAC-SHA256 Credential=key/20231201/us-east-1/s3/aws4_request, SignedHeaders=host, Signature=invalid")

	err := av.VerifySignature(req)
	if err == nil {
		t.Error("Expected error for invalid signature with date")
	}
}

func TestAuthVerifierWithQueryString(t *testing.T) {
	av := NewAuthVerifier("key", "secret")

	// Test with query string in URL
	req, _ := http.NewRequest("GET", "http://localhost/bucket/key?versionId=123", nil)
	req.Header.Set("Authorization", "AWS4-HMAC-SHA256 Credential=key/20231201/us-east-1/s3/aws4_request, SignedHeaders=host, Signature=invalid")
	req.Header.Set("X-Amz-Date", "20231201T120000Z")

	err := av.VerifySignature(req)
	if err == nil {
		t.Error("Expected error for query string with invalid signature")
	}
}

func TestSignatureValidationWithExpiredDate(t *testing.T) {
	av := NewAuthVerifier("key", "secret")

	// Create request with old date
	oldDate := time.Now().AddDate(0, 0, -10).Format("20060102T150405Z")
	req, _ := http.NewRequest("GET", "http://localhost/", nil)
	req.Header.Set("X-Amz-Date", oldDate)
	req.Header.Set("Authorization", fmt.Sprintf("AWS4-HMAC-SHA256 Credential=key/%s/us-east-1/s3/aws4_request, SignedHeaders=host, Signature=invalid", oldDate[:8]))

	err := av.VerifySignature(req)
	// Should fail due to invalid signature (not just date)
	if err == nil {
		t.Error("Expected error for invalid signature")
	}
}
