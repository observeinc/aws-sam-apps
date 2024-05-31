package tracing

import (
	"context"
	"fmt"
	"net/url"
	"os"

	"github.com/observeinc/aws-sam-apps/version"

	"github.com/go-logr/logr"
	"go.opentelemetry.io/contrib/detectors/aws/lambda"
	"go.opentelemetry.io/contrib/exporters/autoexport"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
)

var serviceVersionKey = attribute.Key("service.version")

func SetLogger(logger logr.Logger) {
	otel.SetLogger(logger)
}

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

	options := []resource.Option{
		resource.WithAttributes(serviceVersionKey.String(version.Version)),
		resource.WithFromEnv(),
	}
	if os.Getenv("AWS_LAMBDA_FUNCTION_NAME") != "" {
		options = append(options, resource.WithDetectors(lambda.NewResourceDetector()))
	}

	res, err := resource.New(ctx, options...)
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
