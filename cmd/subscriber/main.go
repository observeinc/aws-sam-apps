package main

import (
	"context"
	"fmt"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/go-logr/logr"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/observeinc/aws-sam-apps/handler"
	"github.com/observeinc/aws-sam-apps/handler/subscriber"
	"github.com/observeinc/aws-sam-apps/logging"
	"github.com/observeinc/aws-sam-apps/tracing"
	"github.com/observeinc/aws-sam-apps/version"
)

const (
	instrumentationName = "github.com/observeinc/aws-sam-apps/cmd/subscriber"
)

var env struct {
	FilterName                  string   `env:"FILTER_NAME"`
	FilterPattern               string   `env:"FILTER_PATTERN"`
	DestinationARN              string   `env:"DESTINATION_ARN"`
	RoleARN                     *string  `env:"ROLE_ARN,noinit"` // noinit retains nil if env var unset
	LogGroupNamePatterns        []string `env:"LOG_GROUP_NAME_PATTERNS"`
	LogGroupNamePrefixes        []string `env:"LOG_GROUP_NAME_PREFIXES"`
	ExcludeLogGroupNamePatterns []string `env:"EXCLUDE_LOG_GROUP_NAME_PATTERNS"`
	QueueURL                    string   `env:"QUEUE_URL,required"`
	Verbosity                   int      `env:"VERBOSITY,default=1"`
	ServiceName                 string   `env:"OTEL_SERVICE_NAME,default=subscriber"`

	AWSMaxAttempts string `env:"AWS_MAX_ATTEMPTS,default=7"`
	AWSRetryMode   string `env:"AWS_RETRY_MODE,default=adaptive"`

	OTELServiceName          string `env:"OTEL_SERVICE_NAME,default=subscriber"`
	OTELTracesExporter       string `env:"OTEL_TRACES_EXPORTER,default=none"`
	OTELExporterOTLPEndpoint string `env:"OTEL_EXPORTER_OTLP_ENDPOINT"`
}

var (
	logger     logr.Logger
	entrypoint lambda.Handler
	options    []lambda.Option
)

func init() {
	if err := realInit(); err != nil {
		panic(err)
	}
}

func realInit() error {
	ctx := context.Background()
	err := handler.ProcessEnv(ctx, &env)
	if err != nil {
		return fmt.Errorf("failed to init: %w", err)
	}

	logger = logging.New(&logging.Config{
		Verbosity: env.Verbosity,
	})
	logger.V(4).Info("initialized", "config", env)

	tracerProvider, err := tracing.NewTracerProvider(ctx)
	if err != nil {
		return fmt.Errorf("failed to initialize tracing: %w", err)
	}

	options = append(options, lambda.WithEnableSIGTERM(func() {
		logger.V(4).Info("SIGTERM received, running shutdown")
		if err := tracerProvider.Shutdown(ctx); err != nil {
			logger.V(4).Error(err, "tracer shutdown failed")
		}
		logger.V(4).Info("shutdown done running")
	}))

	tracer := tracerProvider.Tracer(
		instrumentationName,
		trace.WithInstrumentationVersion(version.Version),
	)

	ctx, span := tracer.Start(ctx, "realInit", trace.WithSpanKind(trace.SpanKindServer))
	defer func() {
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()

	awsCfg, err := tracing.AWSLoadDefaultConfig(ctx, tracerProvider)
	if err != nil {
		return fmt.Errorf("failed to load AWS configuration: %w", err)
	}

	queue, err := subscriber.NewQueue(sqs.NewFromConfig(awsCfg), env.QueueURL)
	if err != nil {
		return fmt.Errorf("failed to load queue: %w", err)
	}

	iq := subscriber.InstrumentQueue(*queue)

	s, err := subscriber.New(&subscriber.Config{
		FilterName:                  env.FilterName,
		FilterPattern:               env.FilterPattern,
		DestinationARN:              env.DestinationARN,
		RoleARN:                     env.RoleARN,
		LogGroupNamePrefixes:        env.LogGroupNamePrefixes,
		LogGroupNamePatterns:        env.LogGroupNamePatterns,
		ExcludeLogGroupNamePatterns: env.ExcludeLogGroupNamePatterns,
		CloudWatchLogsClient:        cloudwatchlogs.NewFromConfig(awsCfg),
		Queue:                       &iq,
	})
	if err != nil {
		return fmt.Errorf("failed to create handler: %w", err)
	}

	is := &subscriber.InstrumentedHandler{
		Handler: s,
		Tracer:  tracer,
	}

	mux := &handler.Mux{
		Logger: logger,
	}

	if err := mux.Register(is.HandleRequest, is.HandleSQS); err != nil {
		return fmt.Errorf("failed to register functions: %w", err)
	}

	entrypoint = &tracing.LambdaHandler{
		Handler: mux,
		Tracer:  tracer,
	}
	return nil
}

func main() {
	lambda.StartWithOptions(entrypoint, options...)
}
