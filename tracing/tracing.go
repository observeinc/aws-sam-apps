package tracing

import (
	"context"
	"fmt"
	"net/url"
	"os"

	"go.opentelemetry.io/contrib/detectors/aws/lambda"
	"go.opentelemetry.io/contrib/exporters/autoexport"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
)

// The OTEL SDK does not handle basic auth in OTEL_EXPORTER_OTLP_ENDPOINT
// Extract username and password and set as OTLP Headers.
func handleOTLPEndpointAuth() error {
	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if endpoint == "" {
		return nil
	}

	if u, err := url.Parse(endpoint); err != nil {
		return fmt.Errorf("failed to parse OTEL_EXPORTER_OTLP_ENDPOINT: %w", err)
	} else if userinfo := u.User; userinfo != nil {
		authHeader := "Bearer " + userinfo.String()

		headers := os.Getenv("OTEL_EXPORTER_OTLP_HEADERS")
		if headers != "" {
			headers += ","
		}
		headers += "Authorization=" + authHeader

		// remove auth from URL
		u.User = nil
		os.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", u.String())
		os.Setenv("OTEL_EXPORTER_OTLP_HEADERS", headers)
	}
	return nil
}

func NewTracerProvider(ctx context.Context) (*trace.TracerProvider, error) {
	if err := handleOTLPEndpointAuth(); err != nil {
		return nil, fmt.Errorf("failed to handle OTLP endpoint auth: %w", err)
	}

	res, err := resource.New(ctx,
		resource.WithDetectors(lambda.NewResourceDetector()),
		resource.WithFromEnv(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	exporter, err := autoexport.NewSpanExporter(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create span exporter: %w", err)
	}

	return trace.NewTracerProvider(
		trace.WithBatcher(exporter),
		trace.WithResource(res),
	), nil
}
