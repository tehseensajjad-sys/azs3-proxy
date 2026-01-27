package azureblob

import (
	"context"
	"testing"
)

func TestRequestIDFromContextPriority(t *testing.T) {
	ctx := context.WithValue(context.Background(), "request_id", "lower")
	ctx = context.WithValue(ctx, "requestID", "upper")
	ctx = context.WithValue(ctx, "X-Request-ID", "header")

	if got := requestIDFromContext(ctx); got != "upper" {
		t.Fatalf("expected priority requestID, got %q", got)
	}

	if got := requestIDFromContext(context.Background()); got != "" {
		t.Fatalf("expected empty request id, got %q", got)
	}
}
