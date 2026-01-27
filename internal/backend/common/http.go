package common

import (
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/log"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.uber.org/zap"
)

// ConfigureSDKLogging sets the Azure SDK log listener and event filters based on log level.
// Returns whether request/response bodies should be included.
func ConfigureSDKLogging(logger *zap.Logger) bool {
	if logger != nil {
		log.SetListener(func(event log.Event, s string) {
			logger.Debug("Azure SDK", zap.String("event", string(event)), zap.String("msg", s))
		})
	}

	includeBody := strings.ToLower(os.Getenv("LOG_LEVEL")) == "debug"
	if includeBody {
		log.SetEvents(log.EventRequest, log.EventResponse)
	} else {
		log.SetEvents()
	}

	return includeBody
}

// NewTransport creates an HTTP transport with tuned connection pooling and optional OTEL instrumentation.
// spanPrefix is prepended to span names when telemetryEnabled is true.
func NewTransport(spanPrefix string, telemetryEnabled bool) http.RoundTripper {
	defaultTransport := http.DefaultTransport.(*http.Transport).Clone()
	defaultTransport.MaxIdleConns = 1000
	defaultTransport.MaxIdleConnsPerHost = 1000
	defaultTransport.MaxConnsPerHost = 0 // Unlimited
	defaultTransport.IdleConnTimeout = 90 * time.Second
	defaultTransport.DisableCompression = true

	var transport http.RoundTripper = defaultTransport
	if telemetryEnabled {
		transport = otelhttp.NewTransport(
			defaultTransport,
			otelhttp.WithSpanNameFormatter(func(operation string, r *http.Request) string {
				return spanPrefix + r.Method + " " + r.URL.Path
			}),
		)
	}

	return transport
}
