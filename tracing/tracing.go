package tracing

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/observeinc/aws-sam-apps/version"

	"github.com/go-logr/logr"
	"go.opentelemetry.io/contrib/detectors/aws/lambda"
	"go.opentelemetry.io/contrib/exporters/autoexport"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
)

var (
	serviceVersionKey              = attribute.Key("service.version")
	allowedResourceAttributeParams = []string{"deployment.environment"}
)

func SetLogger(logger logr.Logger) {
	otel.SetLogger(logger)
}

// The OTEL SDK does not handle basic auth in OTEL_EXPORTER_OTLP_ENDPOINT
// Extract username and password and set as OTLP Headers.
func UpdateOTELEnvVars(getenv func(string) string, setenv func(string, string) error) error {
	endpoint := getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if endpoint == "" {
		return nil
	}

	u, err := url.ParseRequestURI(endpoint)
	if err != nil {
		return fmt.Errorf("failed to parse OTEL_EXPORTER_OTLP_ENDPOINT: %w", err)
	}

	if userinfo := u.User; userinfo != nil {
		authHeader := "Bearer " + userinfo.String()

		headers := getenv("OTEL_EXPORTER_OTLP_HEADERS")
		if headers != "" {
			headers += ","
		}
		headers += "Authorization=" + authHeader

		// remove auth from URL
		u.User = nil
		if err := setenv("OTEL_EXPORTER_OTLP_HEADERS", headers); err != nil {
			return fmt.Errorf("failed to set OTLP headers: %w", err)
		}
	}

	var resourceAttributes []string

	params := u.Query()
	for _, k := range allowedResourceAttributeParams {
		if v := params.Get(k); v != "" {
			resourceAttributes = append(resourceAttributes, fmt.Sprintf("%s=%s", k, v))
			params.Del(k)
		}
	}

	if len(resourceAttributes) > 0 {
		if existing := getenv("OTEL_RESOURCE_ATTRIBUTES"); existing != "" {
			resourceAttributes = append(resourceAttributes, getenv("OTEL_RESOURCE_ATTRIBUTES"))
		}
		u.RawQuery = params.Encode()
		if err := setenv("OTEL_RESOURCE_ATTRIBUTES", strings.Join(resourceAttributes, ",")); err != nil {
			return fmt.Errorf("failed to set OTLP resource attributes: %w", err)
		}
	}

	if getenv("OTEL_EXPORTER_OTLP_ENDPOINT") != u.String() {
		if err := setenv("OTEL_EXPORTER_OTLP_ENDPOINT", u.String()); err != nil {
			return fmt.Errorf("failed to set OTLP endpoint: %w", err)
		}
	}

	return nil
}

func NewTracerProvider(ctx context.Context) (*trace.TracerProvider, error) {
	if err := UpdateOTELEnvVars(os.Getenv, os.Setenv); err != nil {
		return nil, fmt.Errorf("failed to update OTEL environment variables: %w", err)
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
