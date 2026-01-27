package azureblob

import (
	"context"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
)

const clientRequestIDHeader = "x-ms-client-request-id"

// requestIDFromContext extracts a request ID using common context keys.
func requestIDFromContext(ctx context.Context) string {
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

// requestIDPolicy injects x-ms-client-request-id so Azure echoes it back, enabling end-to-end tracing.
type requestIDPolicy struct{}

func (p requestIDPolicy) Do(req *policy.Request) (*http.Response, error) {
	if rid := requestIDFromContext(req.Raw().Context()); rid != "" {
		if req.Raw().Header.Get(clientRequestIDHeader) == "" {
			req.Raw().Header.Set(clientRequestIDHeader, rid)
		}
	}
	return req.Next()
}
