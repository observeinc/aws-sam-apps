package main

import (
	"context"
	"fmt"
	"log"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/go-logr/logr"
	"github.com/sethvargo/go-envconfig"
	lambdadetector "go.opentelemetry.io/contrib/detectors/aws/lambda"
	"go.opentelemetry.io/contrib/exporters/autoexport"
	"go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws"
	"go.opentelemetry.io/otel"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"

	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"go.opentelemetry.io/otel/trace"

	"github.com/observeinc/aws-sam-apps/handler"
	"github.com/observeinc/aws-sam-apps/handler/subscriber"
	"github.com/observeinc/aws-sam-apps/logging"
)

const (
	instrumentationName    = "github.com/observeinc/aws-sam-apps/cmd/subscriber"
	instrumentationVersion = "0.1.0"
)

var tracer = otel.GetTracerProvider().Tracer(
	instrumentationName,
	trace.WithInstrumentationVersion(instrumentationVersion),
)

var env struct {
	FilterName           string   `env:"FILTER_NAME"`
	FilterPattern        string   `env:"FILTER_PATTERN"`
	DestinationARN       string   `env:"DESTINATION_ARN"`
	RoleARN              *string  `env:"ROLE_ARN,noinit"` // noinit retains nil if env var unset
	LogGroupNamePatterns []string `env:"LOG_GROUP_NAME_PATTERNS"`
	LogGroupNamePrefixes []string `env:"LOG_GROUP_NAME_PREFIXES"`
	QueueURL             string   `env:"QUEUE_URL,required"`
	Verbosity            int      `env:"VERBOSITY,default=1"`
}

var (
	logger         logr.Logger
	entrypoint     handler.Mux
	tracerProvider *sdktrace.TracerProvider
)

func init() {
	if err := realInit(); err != nil {
		panic(err)
	}
}

func realInit() error {
	ctx := context.Background()
	detector := lambdadetector.NewResourceDetector()
	res, err := resource.New(ctx,
		resource.WithDetectors(detector),
		resource.WithAttributes(semconv.ServiceName("aws-sam-apps/subscriber")),
	)
	if err != nil {
		panic(err)
	}
	exporter, err := autoexport.NewSpanExporter(ctx)
	if err != nil {
		panic(err)
	}
	tracerProvider = sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tracerProvider)

	err = envconfig.Process(ctx, &env)
	if err != nil {
		return fmt.Errorf("failed to load environment variables: %w", err)
	}

	logger = logging.New(&logging.Config{
		Verbosity: env.Verbosity,
	})

	logger.V(4).Info("initialized", "config", env)

	awsCfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to load AWS configuration: %w", err)
	}

	otelaws.AppendMiddlewares(&awsCfg.APIOptions)

	queue, err := subscriber.NewQueue(sqs.NewFromConfig(awsCfg), env.QueueURL)
	if err != nil {
		return fmt.Errorf("failed to load queue: %w", err)
	}
	s, err := subscriber.New(&subscriber.Config{
		FilterName:           env.FilterName,
		FilterPattern:        env.FilterPattern,
		DestinationARN:       env.DestinationARN,
		RoleARN:              env.RoleARN,
		LogGroupNamePrefixes: env.LogGroupNamePrefixes,
		LogGroupNamePatterns: env.LogGroupNamePatterns,
		Logger:               &logger,
		CloudWatchLogsClient: cloudwatchlogs.NewFromConfig(awsCfg),
		Queue:                queue,
		Tracer:               tracer,
	})
	if err != nil {
		return fmt.Errorf("failed to create handler: %w", err)
	}
	entrypoint.Logger = logger

	is := subscriber.InstrumentHandler(s)

	if err := entrypoint.Register(is.HandleRequest, is.HandleSQS); err != nil {
		return fmt.Errorf("failed to register functions: %w", err)
	}

	return nil
}

func main() {
	ctx := context.Background()
	lambda.StartWithOptions(&entrypoint, lambda.WithEnableSIGTERM(func() {
		log.Printf("SIGTERM received, running shutdown")
		tracerProvider.Shutdown(ctx)
		log.Printf("Shutdown done running")
	}))
}
