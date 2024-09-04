package subscriber

import (
	"context"
	"fmt"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/go-logr/logr"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/observeinc/aws-sam-apps/pkg/handler"
	"github.com/observeinc/aws-sam-apps/pkg/handler/subscriber"
	subscribertracing "github.com/observeinc/aws-sam-apps/pkg/handler/subscriber/tracing"
	"github.com/observeinc/aws-sam-apps/pkg/logging"
	"github.com/observeinc/aws-sam-apps/pkg/tracing"
	"github.com/observeinc/aws-sam-apps/pkg/version"
)

const (
	instrumentationName = "github.com/observeinc/aws-sam-apps/pkg/lambda/subscriber"
)

type Config struct {
	FilterName                  string   `env:"FILTER_NAME"`
	FilterPattern               string   `env:"FILTER_PATTERN"`
	DestinationARN              string   `env:"DESTINATION_ARN"`
	RoleARN                     *string  `env:"ROLE_ARN,noinit"` // noinit retains nil if env var unset
	LogGroupNamePatterns        []string `env:"LOG_GROUP_NAME_PATTERNS"`
	LogGroupNamePrefixes        []string `env:"LOG_GROUP_NAME_PREFIXES"`
	ExcludeLogGroupNamePatterns []string `env:"EXCLUDE_LOG_GROUP_NAME_PATTERNS"`
	QueueURL                    string   `env:"QUEUE_URL,required"`
	ServiceName                 string   `env:"OTEL_SERVICE_NAME,default=subscriber"`

	Logging *logging.Config

	AWSMaxAttempts string `env:"AWS_MAX_ATTEMPTS,default=7"`
	AWSRetryMode   string `env:"AWS_RETRY_MODE,default=adaptive"`

	OTELServiceName          string `env:"OTEL_SERVICE_NAME,default=subscriber"`
	OTELTracesExporter       string `env:"OTEL_TRACES_EXPORTER,default=none"`
	OTELExporterOTLPEndpoint string `env:"OTEL_EXPORTER_OTLP_ENDPOINT"`
}

type Lambda struct {
	Logger     logr.Logger
	Entrypoint lambda.Handler
	Shutdown   func()
}

func New(ctx context.Context, cfg *Config) (*Lambda, error) {
	logger := logging.New(cfg.Logging)
	logger.V(4).Info("initialized", "config", cfg)

	tracing.SetLogger(logger)

	tracerProvider, err := tracing.NewTracerProvider(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize tracing: %w", err)
	}

	l := &Lambda{
		Logger: logger,
		Shutdown: func() {
			logger.V(4).Info("SIGTERM received, running shutdown")
			if err := tracerProvider.Shutdown(ctx); err != nil {
				logger.V(4).Error(err, "tracer shutdown failed")
			}
			logger.V(4).Info("shutdown done running")
		},
	}

	tracer := tracerProvider.Tracer(
		instrumentationName,
		trace.WithInstrumentationVersion(version.Version),
	)

	ctx, span := tracer.Start(ctx, "init", trace.WithSpanKind(trace.SpanKindServer))
	defer func() {
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()

	awsCfg, err := tracing.AWSLoadDefaultConfig(ctx, &tracing.AWSConfig{
		Logger:           logger,
		TracerProvider:   tracerProvider,
		AttributeSetters: subscribertracing.AttributeSetters,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS configuration: %w", err)
	}

	queue, err := subscriber.NewQueue(sqs.NewFromConfig(awsCfg), cfg.QueueURL)
	if err != nil {
		return nil, fmt.Errorf("failed to load queue: %w", err)
	}

	iq := subscriber.InstrumentQueue(*queue)

	s, err := subscriber.New(&subscriber.Config{
		FilterName:                  cfg.FilterName,
		FilterPattern:               cfg.FilterPattern,
		DestinationARN:              cfg.DestinationARN,
		RoleARN:                     cfg.RoleARN,
		LogGroupNamePrefixes:        cfg.LogGroupNamePrefixes,
		LogGroupNamePatterns:        cfg.LogGroupNamePatterns,
		ExcludeLogGroupNamePatterns: cfg.ExcludeLogGroupNamePatterns,
		CloudWatchLogsClient:        cloudwatchlogs.NewFromConfig(awsCfg),
		Queue:                       &iq,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create handler: %w", err)
	}

	is := &subscriber.InstrumentedHandler{
		Handler: s,
		Tracer:  tracer,
	}

	mux := &handler.Mux{
		Logger: logger,
	}

	if err := mux.Register(is.HandleRequest, is.HandleSQS, is.HandleCloudFormation); err != nil {
		return nil, fmt.Errorf("failed to register functions: %w", err)
	}

	l.Entrypoint = tracing.WrapHandlerSQSContext(tracing.NewLambdaHandler(mux, tracerProvider))
	return l, nil
}
