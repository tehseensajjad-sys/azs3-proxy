package azureblob

import (
	"context"
	"net/http"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"go.uber.org/zap"

	backendcommon "github.com/vibhansa-msft/azs3-proxy/internal/backend/common"
)

func TestRecordAzureRequestCapturesIDs(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	backend := &AzureBlobBackend{logger: logger}

	ctx := context.WithValue(context.Background(), "requestID", "s3-req-123")

	resp := &http.Response{Header: http.Header{}}
	resp.Header.Set("x-ms-request-id", "azure-req-id")
	resp.Header.Set(backendcommon.ClientRequestIDHeader, "client-req-id")

	err := &azcore.ResponseError{RawResponse: resp}

	backend.recordAzureRequest(ctx, "TestOperation", err)
}
