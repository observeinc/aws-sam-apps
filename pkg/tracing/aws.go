package tracing

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/middleware"
	"github.com/aws/aws-sdk-go-v2/config"
	"go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws"
	"go.opentelemetry.io/otel/trace"

	"github.com/observeinc/aws-sam-apps/pkg/version"
)

func AWSLoadDefaultConfig(ctx context.Context, tracerProvider trace.TracerProvider) (aws.Config, error) {
	awsCfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return awsCfg, fmt.Errorf("failed to load AWS configuration: %w", err)
	}

	if serviceName := os.Getenv("OTEL_SERVICE_NAME"); serviceName != "" {
		awsCfg.APIOptions = append(awsCfg.APIOptions,
			middleware.AddUserAgentKeyValue(serviceName, version.Version),
		)
	}

	otelaws.AppendMiddlewares(&awsCfg.APIOptions, otelaws.WithTracerProvider(tracerProvider))
	return awsCfg, nil
}
