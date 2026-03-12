package common

import (
	"context"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"

	"github.com/vibhansa-msft/azs3-proxy/internal/version"
)

const ClientRequestIDHeader = "x-ms-client-request-id"

// RequestIDFromContext extracts a request ID using common context keys shared across handlers.
func RequestIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if rid, ok := ctx.Value("requestID").(string); ok && rid != "" {
		return rid
	}
	if rid, ok := ctx.Value("request_id").(string); ok && rid != "" {
		return rid
	}
	if rid, ok := ctx.Value("X-Request-ID").(string); ok && rid != "" {
		return rid
	}
	return ""
}

// userAgentPrefix is computed once at init to avoid repeated string concatenation per request.
var userAgentPrefix = version.AzureApplicationIDPrefix + version.Version

// RequestIDPolicy injects x-ms-client-request-id so Azure echoes it back for correlation
// and prepends the custom User-Agent to bypass the SDK's 24-character ApplicationID limit.
type RequestIDPolicy struct{}

func (p RequestIDPolicy) Do(req *policy.Request) (*http.Response, error) {
	if rid := RequestIDFromContext(req.Raw().Context()); rid != "" {
		if req.Raw().Header.Get(ClientRequestIDHeader) == "" {
			req.Raw().Header.Set(ClientRequestIDHeader, rid)
		}
	}

	// Prepend our full application identifier to the User-Agent header.
	// This bypasses the Azure SDK's 24-character truncation on ApplicationID.
	if ua := req.Raw().Header.Get("User-Agent"); ua == "" {
		req.Raw().Header.Set("User-Agent", userAgentPrefix)
	} else {
		req.Raw().Header.Set("User-Agent", userAgentPrefix+" "+ua)
	}

	return req.Next()
}
