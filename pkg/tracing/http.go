package tracing

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/go-logr/logr"
	"github.com/hashicorp/go-retryablehttp"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/trace"

	"github.com/observeinc/aws-sam-apps/pkg/version"
)

// leveledLogger provides an adapter between logr.Logger and retryablehttp.LeveledLogger.
type leveledLogger struct {
	logr.Logger
}

func (l *leveledLogger) Error(msg string, keysAndValues ...interface{}) {
	l.V(1).Info(msg, keysAndValues...)
}

func (l *leveledLogger) Warn(msg string, keysAndValues ...interface{}) {
	l.V(2).Info(msg, keysAndValues...)
}

func (l *leveledLogger) Info(msg string, keysAndValues ...interface{}) {
	l.V(3).Info(msg, keysAndValues...)
}

func (l *leveledLogger) Debug(msg string, keysAndValues ...interface{}) {
	l.V(4).Info(msg, keysAndValues...)
}

type HTTPClientConfig struct {
	RetryWaitMin   *time.Duration // Minimum time to wait on retry
	RetryWaitMax   *time.Duration // Maximumum time to wait on retry
	RetryMax       *int           // Maximum number of retries
	HTTPClient     *http.Client
	Logger         *logr.Logger
	TracerProvider trace.TracerProvider
}

func NewHTTPClient(cfg *HTTPClientConfig) *http.Client {
	if cfg == nil {
		cfg = &HTTPClientConfig{}
	}

	client := retryablehttp.NewClient()

	if cfg.HTTPClient != nil {
		client.HTTPClient = cfg.HTTPClient
	}

	if cfg.RetryWaitMin != nil {
		client.RetryWaitMin = *cfg.RetryWaitMin
	}

	if cfg.RetryWaitMax != nil {
		client.RetryWaitMax = *cfg.RetryWaitMax
	}

	if cfg.RetryMax != nil {
		client.RetryMax = *cfg.RetryMax
	}

	logger := logr.Discard()
	if cfg.Logger != nil {
		logger = *cfg.Logger
	}

	client.Logger = &leveledLogger{logger}

	var transport http.RoundTripper = &retryablehttp.RoundTripper{Client: client}
	if serviceName := os.Getenv("OTEL_SERVICE_NAME"); serviceName != "" {
		transport = &addUserAgent{
			RoundTripper: transport,
			UserAgent:    fmt.Sprintf("%s/%s", serviceName, version.Version),
		}
	}
	return &http.Client{
		Transport: otelhttp.NewTransport(transport,
			otelhttp.WithTracerProvider(cfg.TracerProvider),
			otelhttp.WithSpanOptions(
				trace.WithSpanKind(trace.SpanKindClient),
			),
		),
	}
}

type addUserAgent struct {
	UserAgent string
	http.RoundTripper
}

func (rt *addUserAgent) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Add("User-Agent", rt.UserAgent)
	//nolint:wrapcheck
	return rt.RoundTripper.RoundTrip(req)
}
