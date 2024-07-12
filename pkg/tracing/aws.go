package tracing

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	v2middleware "github.com/aws/aws-sdk-go-v2/aws/middleware"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/smithy-go/logging"
	"github.com/go-logr/logr"
	"go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws"
	"go.opentelemetry.io/otel/trace"

	"github.com/observeinc/aws-sam-apps/pkg/version"
)

type AWSConfig struct {
	Logger           logr.Logger
	TracerProvider   trace.TracerProvider
	AttributeSetters []otelaws.AttributeSetter
}

func AWSLoadDefaultConfig(ctx context.Context, cfg *AWSConfig) (aws.Config, error) {
	awsCfg, err := config.LoadDefaultConfig(ctx,
		config.WithLogger(logging.LoggerFunc(func(classification logging.Classification, format string, v ...interface{}) {
			switch classification {
			case logging.Debug:
				cfg.Logger.V(4).Info(fmt.Sprintf(format, v...))
			case logging.Warn:
				cfg.Logger.Info(fmt.Sprintf(format, v...))
			}
		})),
	)
	if err != nil {
		return awsCfg, fmt.Errorf("failed to load AWS configuration: %w", err)
	}

	switch {
	case cfg.Logger.V(8).Enabled():
		awsCfg.ClientLogMode |= aws.LogRequestWithBody | aws.LogResponseWithBody
	case cfg.Logger.V(6).Enabled():
		awsCfg.ClientLogMode |= aws.LogRequest | aws.LogResponse
	}

	if serviceName := os.Getenv("OTEL_SERVICE_NAME"); serviceName != "" {
		awsCfg.APIOptions = append(awsCfg.APIOptions,
			v2middleware.AddUserAgentKeyValue(serviceName, version.Version),
		)
	}

	otelaws.AppendMiddlewares(&awsCfg.APIOptions,
		otelaws.WithTracerProvider(cfg.TracerProvider),
		otelaws.WithAttributeSetter(append(cfg.AttributeSetters, otelaws.DefaultAttributeSetter)...),
	)
	return awsCfg, nil
}
