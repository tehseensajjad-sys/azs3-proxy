package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"
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
	// TODO: Implement full SigV4 verification
	// 1. Parse Authorization header
	// 2. Extract credential scope
	// 3. Calculate string to sign
	// 4. Verify signature
	return nil
}

// calculateSignature calculates AWS SigV4 signature
func (av *AuthVerifier) calculateSignature(stringToSign string, dateStamp string) string {
	// TODO: Implement signature calculation
	// kSecret = "AWS4" + secretKey
	// kDate = HMAC-SHA256(kSecret, "YYYYMMDD")
	// kRegion = HMAC-SHA256(kDate, "us-east-1")
	// kService = HMAC-SHA256(kRegion, "s3")
	// kSigning = HMAC-SHA256(kService, "aws4_request")
	// signature = Hex(HMAC-SHA256(kSigning, stringToSign))
	return ""
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
