package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
)

// AuthVerifier handles AWS SigV4 signature verification for incoming S3 API requests.
// It validates that requests have been properly signed with the configured S3 credentials.
type AuthVerifier struct {
	accessKey string // S3 access key used to verify request signatures
	secretKey string // S3 secret key used to compute request signatures
}

// NewAuthVerifier creates a new AWS SigV4 signature verifier with the provided S3 credentials.
// Both accessKey and secretKey are required for signature verification.
func NewAuthVerifier(accessKey, secretKey string) *AuthVerifier {
	return &AuthVerifier{
		accessKey: accessKey,
		secretKey: secretKey,
	}
}

// VerifySignature verifies that an HTTP request has a valid AWS SigV4 authorization signature.
// It checks the Authorization header format, parses the credential scope, and validates the signature
// matches what would be computed for this request.
// Returns error if the signature is invalid or missing required headers.
//
// Authorization header format:
// AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20130524/us-east-1/s3/aws4_request,
// SignedHeaders=host;range;x-amz-date, Signature=fe5f80f77d5fa3beca038a248ff027d0445342fe2855ddc963176630326f1024
func (av *AuthVerifier) VerifySignature(r *http.Request) error {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return fmt.Errorf("missing authorization header")
	}

	// Check authorization header uses AWS4-HMAC-SHA256 scheme
	if !strings.HasPrefix(authHeader, "AWS4-HMAC-SHA256 ") {
		return fmt.Errorf("invalid authorization header format")
	}

	// Parse authorization header into three parts: Credential, SignedHeaders, Signature
	parts := strings.Split(authHeader[17:], ", ")
	if len(parts) != 3 {
		return fmt.Errorf("invalid authorization header parts")
	}

	// Extract each component from the authorization header
	credentialPart := strings.TrimPrefix(parts[0], "Credential=")
	signedHeadersPart := strings.TrimPrefix(parts[1], "SignedHeaders=")
	signaturePart := strings.TrimPrefix(parts[2], "Signature=")

	// Parse the credential scope (format: AccessKey/Date/Region/Service/aws4_request)
	credentialParts := strings.Split(credentialPart, "/")
	if len(credentialParts) != 5 {
		return fmt.Errorf("invalid credential scope")
	}

	// Extract credential scope components
	accessKey := credentialParts[0]
	dateStamp := credentialParts[1]
	region := credentialParts[2]
	service := credentialParts[3]
	awsRequest := credentialParts[4]

	// Verify the access key matches our configured credentials
	if accessKey != av.accessKey {
		return fmt.Errorf("invalid access key")
	}

	// Verify service is "s3" and request type is "aws4_request"
	if service != "s3" || awsRequest != "aws4_request" {
		return fmt.Errorf("invalid service or request type")
	}

	// Build the canonical request representation for signature computation
	canonicalRequest := av.buildCanonicalRequest(r, signedHeadersPart)

	// Get the timestamp header required for SigV4
	amzDate := r.Header.Get("X-Amz-Date")
	if amzDate == "" {
		return fmt.Errorf("missing X-Amz-Date header")
	}

	// Build the string to sign (includes algorithm, timestamp, credential scope, and request hash)
	stringToSign := av.buildStringToSign(canonicalRequest, amzDate, dateStamp, region, service)

	// Calculate expected signature from the request data
	expectedSignature := av.calculateSignature(stringToSign, dateStamp)

	// Verify the signature matches what client sent
	if expectedSignature != signaturePart {
		return fmt.Errorf("signature mismatch")
	}

	return nil
}

// buildCanonicalRequest builds the canonical request representation required for signature calculation.
// The canonical request is a standardized string representation of the HTTP request that is then hashed.
// Format:
//
//	HTTPMethod + '\n' +
//	CanonicalURI + '\n' +
//	CanonicalQueryString + '\n' +
//	CanonicalHeaders + '\n' +
//	SignedHeaders + '\n' +
//	HashedPayload
func (av *AuthVerifier) buildCanonicalRequest(r *http.Request, signedHeaders string) string {
	// Get HTTP method (GET, PUT, POST, DELETE, HEAD)
	method := r.Method

	// Get request URI, default to "/" if empty
	canonicalURI := r.URL.Path
	if canonicalURI == "" {
		canonicalURI = "/"
	}

	// Get query string from request URL (if present)
	canonicalQueryString := r.URL.RawQuery

	// Build canonical headers string from signed headers list
	canonicalHeaders := av.buildCanonicalHeaders(r, signedHeaders)

	// Hashed payload - for GET requests without body, use hash of empty string
	hashedPayload := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855" // SHA256("")

	// Combine all components into canonical request format
	canonicalRequest := fmt.Sprintf("%s\n%s\n%s\n%s\n%s\n%s",
		method,
		canonicalURI,
		canonicalQueryString,
		canonicalHeaders,
		signedHeaders,
		hashedPayload,
	)

	return canonicalRequest
}

// buildCanonicalHeaders builds the canonical headers string for signature calculation.
// Headers are formatted as "name:value" pairs, sorted by header name, with trailing newline.
func (av *AuthVerifier) buildCanonicalHeaders(r *http.Request, signedHeaders string) string {
	// Parse list of signed header names (e.g., "host;x-amz-content-sha256;x-amz-date")
	headerNames := strings.Split(signedHeaders, ";")
	var headers []string

	// Extract and format each signed header
	for _, name := range headerNames {
		headerName := strings.TrimSpace(name)
		// Get header value, fallback to lowercase variant if not found
		headerValue := r.Header.Get(headerName)
		if headerValue == "" {
			headerValue = r.Header.Get(strings.ToLower(headerName))
		}

		// Append formatted header (lowercase name with value)
		headers = append(headers, fmt.Sprintf("%s:%s", strings.ToLower(headerName), strings.TrimSpace(headerValue)))
	}

	// Join headers with newlines and add trailing newline
	return strings.Join(headers, "\n") + "\n"
}

// buildStringToSign builds the string to sign for AWS SigV4 signature calculation.
// This string includes the algorithm, timestamp, credential scope, and hashed canonical request.
// Format:
//
//	AWS4-HMAC-SHA256\n +
//	timestamp\n +
//	credential_scope\n +
//	hashed_canonical_request
func (av *AuthVerifier) buildStringToSign(canonicalRequest, amzDate, dateStamp, region, service string) string {
	// Hash the canonical request using SHA256
	hashedCanonicalRequest := hashSHA256(canonicalRequest)

	// Build string to sign with algorithm, timestamp, credential scope, and request hash
	return fmt.Sprintf("AWS4-HMAC-SHA256\n%s\n%s/%s/%s/aws4_request\n%s",
		amzDate,
		dateStamp,
		region,
		service,
		hashedCanonicalRequest,
	)
}

// calculateSignature calculates the AWS SigV4 signature using the derived signing key.
// SigV4 uses a multi-step HMAC derivation process:
//
//	kSecret = "AWS4" + secretAccessKey
//	kDate = HMAC-SHA256(kSecret, "YYYYMMDD")
//	kRegion = HMAC-SHA256(kDate, "region")
//	kService = HMAC-SHA256(kRegion, "s3")
//	kSigning = HMAC-SHA256(kService, "aws4_request")
//	signature = Hex(HMAC-SHA256(kSigning, stringToSign))
func (av *AuthVerifier) calculateSignature(stringToSign string, dateStamp string) string {
	// Start with AWS4 prefix and secret key
	kSecret := "AWS4" + av.secretKey

	// Derive key for date
	kDate := hmacSHA256(kSecret, dateStamp)

	// Derive key for region (always us-east-1 for S3)
	kRegion := hmacSHA256(string(kDate), "us-east-1")

	// Derive key for service (s3)
	kService := hmacSHA256(string(kRegion), "s3")

	// Derive signing key
	kSigning := hmacSHA256(string(kService), "aws4_request")

	// Sign the string to sign with the derived key
	signature := hmacSHA256(string(kSigning), stringToSign)

	// Return hex-encoded signature
	return hex.EncodeToString(signature)
}

// hmacSHA256 computes HMAC-SHA256 hash of data with the provided key.
// Returns raw bytes that can be used as key material for next HMAC operation.
func hmacSHA256(key, data string) []byte {
	h := hmac.New(sha256.New, []byte(key))
	h.Write([]byte(data))
	return h.Sum(nil)
}

// hashSHA256 computes SHA256 hash of data and returns hex-encoded string.
// Used for hashing the canonical request in SigV4 calculation.
func hashSHA256(data string) string {
	h := sha256.New()
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}
