package forwarder

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/go-logr/logr"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/observeinc/aws-sam-apps/pkg/handler"
	"github.com/observeinc/aws-sam-apps/pkg/handler/forwarder"
	"github.com/observeinc/aws-sam-apps/pkg/handler/forwarder/override"
	"github.com/observeinc/aws-sam-apps/pkg/handler/forwarder/s3http"
	"github.com/observeinc/aws-sam-apps/pkg/logging"
	"github.com/observeinc/aws-sam-apps/pkg/tracing"
	"github.com/observeinc/aws-sam-apps/pkg/version"
)

const (
	instrumentationName = "github.com/observeinc/aws-sam-apps/pkg/lambda/forwarder"
)

type S3Client interface {
	GetObject(context.Context, *s3.GetObjectInput, ...func(*s3.Options)) (*s3.GetObjectOutput, error)
	PutObject(context.Context, *s3.PutObjectInput, ...func(*s3.Options)) (*s3.PutObjectOutput, error)
	CopyObject(context.Context, *s3.CopyObjectInput, ...func(*s3.Options)) (*s3.CopyObjectOutput, error)
	s3.HeadBucketAPIClient
}

type Config struct {
	DestinationURI       string           `env:"DESTINATION_URI,required"`
	MaxFileSize          int64            `env:"MAX_FILE_SIZE"`
	ContentTypeOverrides []*override.Rule `env:"CONTENT_TYPE_OVERRIDES"`
	PresetOverrides      []string         `env:"PRESET_OVERRIDES,default=aws/v1,infer/v1"`
	SourceBucketNames    []string         `env:"SOURCE_BUCKET_NAMES"`

	Logging *logging.Config

	OTELServiceName          string `env:"OTEL_SERVICE_NAME,default=forwarder"`
	OTELTracesExporter       string `env:"OTEL_TRACES_EXPORTER,default=none"`
	OTELExporterOTLPEndpoint string `env:"OTEL_EXPORTER_OTLP_ENDPOINT"`

	S3HTTPGzipLevel *int `env:"S3_HTTP_GZIP_LEVEL,default=1"`

	// The following variables are not configurable via environment
	HTTPInsecureSkipVerify bool     `json:"-"`
	AWSS3Client            S3Client `json:"-"`
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

	awsCfg, err := tracing.AWSLoadDefaultConfig(ctx, tracerProvider)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS configuration: %w", err)
	}

	customOverrides := &override.Set{
		Logger: logger.WithValues("set", "custom"),
		Rules:  cfg.ContentTypeOverrides,
	}
	if err := customOverrides.Validate(); err != nil {
		return nil, fmt.Errorf("failed to validate override set: %w", err)
	}

	presets, err := override.LoadPresets(logger, cfg.PresetOverrides...)
	if err != nil {
		return nil, fmt.Errorf("failed to load presets: %w", err)
	}

	var awsS3Client = cfg.AWSS3Client
	if awsS3Client == nil {
		awsS3Client = s3.NewFromConfig(awsCfg)
	}

	var s3Client forwarder.S3Client = awsS3Client
	if strings.HasPrefix(cfg.DestinationURI, "https") {
		logger.V(4).Info("loading http client")
		s3Client, err = s3http.New(&s3http.Config{
			DestinationURI:     cfg.DestinationURI,
			GetObjectAPIClient: awsS3Client,
			GzipLevel:          cfg.S3HTTPGzipLevel,
			HTTPClient: tracing.NewHTTPClient(&tracing.HTTPClientConfig{
				TracerProvider:     tracerProvider,
				Logger:             &logger,
				InsecureSkipVerify: cfg.HTTPInsecureSkipVerify,
			}),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to load http client: %w", err)
		}
	}

	f, err := forwarder.New(&forwarder.Config{
		DestinationURI:    cfg.DestinationURI,
		MaxFileSize:       cfg.MaxFileSize,
		S3Client:          s3Client,
		Override:          append(override.Sets{customOverrides}, presets...),
		SourceBucketNames: cfg.SourceBucketNames,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create handler: %w", err)
	}

	region, err := f.GetDestinationRegion(ctx, awsS3Client)
	if err != nil {
		return nil, fmt.Errorf("failed to get destination region: %w", err)
	}

	if region != "" && awsCfg.Region != region {
		logger.V(4).Info("modifying s3 client region", "region", region)
		regionCfg := awsCfg.Copy()
		regionCfg.Region = region
		f.S3Client = s3.NewFromConfig(regionCfg)
	}

	mux := &handler.Mux{
		Logger: logger,
	}

	if err := mux.Register(f.Handle); err != nil {
		return nil, fmt.Errorf("failed to register functions: %w", err)
	}

	l.Entrypoint = tracing.NewLambdaHandler(mux, tracerProvider)
	return l, nil
}
