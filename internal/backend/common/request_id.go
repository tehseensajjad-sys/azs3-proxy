package common

import (
	"context"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
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

// RequestIDPolicy injects x-ms-client-request-id so Azure echoes it back for correlation.
type RequestIDPolicy struct{}

func (p RequestIDPolicy) Do(req *policy.Request) (*http.Response, error) {
	if rid := RequestIDFromContext(req.Raw().Context()); rid != "" {
		if req.Raw().Header.Get(ClientRequestIDHeader) == "" {
			req.Raw().Header.Set(ClientRequestIDHeader, rid)
		}
	}
	return req.Next()
}
