package common

import (
	"net/http"
	"testing"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestConfigureSDKLogging(t *testing.T) {
	core, _ := observer.New(zap.DebugLevel)
	logger := zap.New(core)

	t.Setenv("LOG_LEVEL", "debug")
	includeBody := ConfigureSDKLogging(logger)
	if !includeBody {
		t.Fatalf("expected request/response bodies enabled for debug level")
	}

	t.Setenv("LOG_LEVEL", "info")
	includeBody = ConfigureSDKLogging(logger)
	if includeBody {
		t.Fatalf("expected bodies disabled for non-debug level")
	}
}

func TestNewTransport(t *testing.T) {
	tr := NewTransport("", false)
	httpTr, ok := tr.(*http.Transport)
	if !ok {
		t.Fatalf("expected base transport when telemetry disabled")
	}
	if httpTr.MaxIdleConns != 1000 || httpTr.MaxIdleConnsPerHost != 1000 || httpTr.DisableCompression != true {
		t.Fatalf("transport tuning not applied: %+v", httpTr)
	}

	teleTr := NewTransport("s3-proxy:", true)
	if _, ok := teleTr.(*otelhttp.Transport); !ok {
		t.Fatalf("expected otel transport when telemetry enabled")
	}
}
