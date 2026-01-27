package common

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
)

func TestRequestIDFromContextPriority(t *testing.T) {
	ctx := context.WithValue(context.Background(), "request_id", "lower")
	ctx = context.WithValue(ctx, "requestID", "upper")
	ctx = context.WithValue(ctx, "X-Request-ID", "header")

	if got := RequestIDFromContext(ctx); got != "upper" {
		t.Fatalf("expected priority requestID, got %q", got)
	}

	if got := RequestIDFromContext(context.Background()); got != "" {
		t.Fatalf("expected empty request id, got %q", got)
	}
}

type recordingTransport struct {
	req *http.Request
}

func (rt *recordingTransport) Do(req *http.Request) (*http.Response, error) {
	rt.req = req
	return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(""))}, nil
}

func TestRequestIDPolicySetsHeader(t *testing.T) {
	ctx := context.WithValue(context.Background(), "requestID", "abc123")
	transport := &recordingTransport{}
	pipeline := runtime.NewPipeline("module", "v1", runtime.PipelineOptions{
		PerRetry: []policy.Policy{RequestIDPolicy{}},
	}, &policy.ClientOptions{Transport: transport})

	req, err := runtime.NewRequest(ctx, http.MethodGet, "https://example.com")
	if err != nil {
		t.Fatalf("failed to build request: %v", err)
	}

	if _, err := pipeline.Do(req); err != nil {
		t.Fatalf("pipeline returned error: %v", err)
	}

	if got := transport.req.Header.Get(ClientRequestIDHeader); got != "abc123" {
		t.Fatalf("expected header set from context, got %q", got)
	}

	ctx = context.WithValue(context.Background(), "requestID", "abc123")
	req, err = runtime.NewRequest(ctx, http.MethodGet, "https://example.com")
	if err != nil {
		t.Fatalf("failed to build request: %v", err)
	}
	req.Raw().Header.Set(ClientRequestIDHeader, "pre-set")

	if _, err := pipeline.Do(req); err != nil {
		t.Fatalf("pipeline returned error: %v", err)
	}

	if got := transport.req.Header.Get(ClientRequestIDHeader); got != "pre-set" {
		t.Fatalf("expected existing header preserved, got %q", got)
	}
}
