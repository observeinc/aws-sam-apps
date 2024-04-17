package main

import (
	"context"
	"fmt"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/go-logr/logr"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/observeinc/aws-sam-apps/handler"
	"github.com/observeinc/aws-sam-apps/handler/forwarder"
	"github.com/observeinc/aws-sam-apps/handler/forwarder/override"
	"github.com/observeinc/aws-sam-apps/logging"
	"github.com/observeinc/aws-sam-apps/tracing"
	"github.com/observeinc/aws-sam-apps/version"
)

const (
	instrumentationName = "github.com/observeinc/aws-sam-apps/cmd/forwarder"
)

var env struct {
	DestinationURI       string           `env:"DESTINATION_URI,required"`
	Verbosity            int              `env:"VERBOSITY,default=1"`
	MaxFileSize          int64            `env:"MAX_FILE_SIZE"`
	ContentTypeOverrides []*override.Rule `env:"CONTENT_TYPE_OVERRIDES"`
	PresetOverrides      []string         `env:"PRESET_OVERRIDES,default=aws/v1"`
	SourceBucketNames    []string         `env:"SOURCE_BUCKET_NAMES"`

	OTELServiceName          string `env:"OTEL_SERVICE_NAME,default=forwarder"`
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

func realInit() (err error) {
	ctx := context.Background()

	err = handler.ProcessEnv(ctx, &env)
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

	ctx, span := tracer.Start(ctx, "realInit")
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

	customOverrides := &override.Set{
		Logger: logger.WithValues("set", "custom"),
		Rules:  env.ContentTypeOverrides,
	}
	if err := customOverrides.Validate(); err != nil {
		return fmt.Errorf("failed to validate override set: %w", err)
	}

	presets, err := override.LoadPresets(logger, env.PresetOverrides...)
	if err != nil {
		return fmt.Errorf("failed to load presets: %w", err)
	}

	s3client := s3.NewFromConfig(awsCfg)

	f, err := forwarder.New(&forwarder.Config{
		DestinationURI:    env.DestinationURI,
		MaxFileSize:       env.MaxFileSize,
		S3Client:          s3client,
		Override:          append(override.Sets{customOverrides}, presets...),
		SourceBucketNames: env.SourceBucketNames,
	})
	if err != nil {
		return fmt.Errorf("failed to create handler: %w", err)
	}

	region, err := f.GetDestinationRegion(ctx, s3client)
	if err != nil {
		return fmt.Errorf("failed to get destination region: %w", err)
	}

	if awsCfg.Region != region {
		logger.V(4).Info("modifying s3 client region", "region", region)
		regionCfg := awsCfg.Copy()
		regionCfg.Region = region
		f.S3Client = s3.NewFromConfig(regionCfg)
	}

	mux := &handler.Mux{
		Logger: logger,
	}

	if err := mux.Register(f.Handle); err != nil {
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
