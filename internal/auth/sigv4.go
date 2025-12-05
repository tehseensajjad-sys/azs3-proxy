package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
)

type AuthVerifier struct {
	accessKey string
	secretKey string
}

func NewAuthVerifier(accessKey, secretKey string) *AuthVerifier {
	return &AuthVerifier{
		accessKey: accessKey,
		secretKey: secretKey,
	}
}

// VerifySignature verifies AWS SigV4 authorization header
func (av *AuthVerifier) VerifySignature(r *http.Request) error {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return fmt.Errorf("missing authorization header")
	}

	// Parse Authorization header format:
	// AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20130524/us-east-1/s3/aws4_request, SignedHeaders=host;range;x-amz-date, Signature=fe5f80f77d5fa3beca038a248ff027d0445342fe2855ddc963176630326f1024

	if !strings.HasPrefix(authHeader, "AWS4-HMAC-SHA256 ") {
		return fmt.Errorf("invalid authorization header format")
	}

	parts := strings.Split(authHeader[17:], ", ")
	if len(parts) != 3 {
		return fmt.Errorf("invalid authorization header parts")
	}

	// Extract credential, signedHeaders, and signature
	credentialPart := strings.TrimPrefix(parts[0], "Credential=")
	signedHeadersPart := strings.TrimPrefix(parts[1], "SignedHeaders=")
	signaturePart := strings.TrimPrefix(parts[2], "Signature=")

	// Parse credential scope
	credentialParts := strings.Split(credentialPart, "/")
	if len(credentialParts) != 5 {
		return fmt.Errorf("invalid credential scope")
	}

	accessKey := credentialParts[0]
	dateStamp := credentialParts[1]
	region := credentialParts[2]
	service := credentialParts[3]
	awsRequest := credentialParts[4]

	// Verify access key
	if accessKey != av.accessKey {
		return fmt.Errorf("invalid access key")
	}

	// Verify service and request type
	if service != "s3" || awsRequest != "aws4_request" {
		return fmt.Errorf("invalid service or request type")
	}

	// Build canonical request
	canonicalRequest := av.buildCanonicalRequest(r, signedHeadersPart)

	// Build string to sign
	amzDate := r.Header.Get("X-Amz-Date")
	if amzDate == "" {
		return fmt.Errorf("missing X-Amz-Date header")
	}

	stringToSign := av.buildStringToSign(canonicalRequest, amzDate, dateStamp, region, service)

	// Calculate signature
	expectedSignature := av.calculateSignature(stringToSign, dateStamp)

	// Compare signatures
	if expectedSignature != signaturePart {
		return fmt.Errorf("signature mismatch")
	}

	return nil
}

// buildCanonicalRequest builds the canonical request for signature calculation
func (av *AuthVerifier) buildCanonicalRequest(r *http.Request, signedHeaders string) string {
	// CanonicalRequest format:
	// HTTPMethod + '\n' +
	// CanonicalURI + '\n' +
	// CanonicalQueryString + '\n' +
	// CanonicalHeaders + '\n' +
	// SignedHeaders + '\n' +
	// HashedPayload

	method := r.Method
	canonicalURI := r.URL.Path
	if canonicalURI == "" {
		canonicalURI = "/"
	}

	canonicalQueryString := r.URL.RawQuery
	canonicalHeaders := av.buildCanonicalHeaders(r, signedHeaders)
	hashedPayload := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855" // SHA256("")

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

// buildCanonicalHeaders builds the canonical headers for signature calculation
func (av *AuthVerifier) buildCanonicalHeaders(r *http.Request, signedHeaders string) string {
	headerNames := strings.Split(signedHeaders, ";")
	var headers []string

	for _, name := range headerNames {
		headerName := strings.TrimSpace(name)
		headerValue := r.Header.Get(headerName)
		if headerValue == "" {
			// Try with canonical header case
			headerValue = r.Header.Get(strings.ToLower(headerName))
		}
		headers = append(headers, fmt.Sprintf("%s:%s", strings.ToLower(headerName), strings.TrimSpace(headerValue)))
	}

	return strings.Join(headers, "\n") + "\n"
}

// buildStringToSign builds the string to sign for signature calculation
func (av *AuthVerifier) buildStringToSign(canonicalRequest, amzDate, dateStamp, region, service string) string {
	hashedCanonicalRequest := hashSHA256(canonicalRequest)
	return fmt.Sprintf("AWS4-HMAC-SHA256\n%s\n%s/%s/%s/aws4_request\n%s",
		amzDate,
		dateStamp,
		region,
		service,
		hashedCanonicalRequest,
	)
}

// calculateSignature calculates AWS SigV4 signature
func (av *AuthVerifier) calculateSignature(stringToSign string, dateStamp string) string {
	// kSecret = "AWS4" + secretKey
	// kDate = HMAC-SHA256(kSecret, "YYYYMMDD")
	// kRegion = HMAC-SHA256(kDate, "us-east-1")
	// kService = HMAC-SHA256(kRegion, "s3")
	// kSigning = HMAC-SHA256(kService, "aws4_request")
	// signature = Hex(HMAC-SHA256(kSigning, stringToSign))

	kSecret := "AWS4" + av.secretKey
	kDate := hmacSHA256(kSecret, dateStamp)
	kRegion := hmacSHA256(string(kDate), "us-east-1")
	kService := hmacSHA256(string(kRegion), "s3")
	kSigning := hmacSHA256(string(kService), "aws4_request")
	signature := hmacSHA256(string(kSigning), stringToSign)

	return hex.EncodeToString(signature)
}

func hmacSHA256(key, data string) []byte {
	h := hmac.New(sha256.New, []byte(key))
	h.Write([]byte(data))
	return h.Sum(nil)
}

func hashSHA256(data string) string {
	h := sha256.New()
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}
