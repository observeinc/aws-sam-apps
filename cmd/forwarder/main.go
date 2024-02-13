package main

import (
	"context"
	"fmt"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/go-logr/logr"
	"github.com/sethvargo/go-envconfig"

	"github.com/observeinc/aws-sam-apps/handler"
	"github.com/observeinc/aws-sam-apps/handler/forwarder"
	"github.com/observeinc/aws-sam-apps/logging"
)

var env struct {
	DestinationURI       string   `env:"DESTINATION_URI,required"`
	LogPrefix            string   `env:"LOG_PREFIX,default=forwarder/"`
	Verbosity            int      `env:"VERBOSITY,default=1"`
	MaxFileSize          int64    `env:"MAX_FILE_SIZE"`
	ContentTypeOverrides []string `env:"CONTENT_TYPE_OVERRIDES"`
	SourceBucketNames    []string `env:"SOURCE_BUCKET_NAMES"`
}

var (
	logger     logr.Logger
	entrypoint handler.Mux
)

func init() {
	if err := realInit(); err != nil {
		panic(err)
	}
}

func realInit() error {
	ctx := context.Background()

	err := envconfig.Process(ctx, &env)
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

	s3client := s3.NewFromConfig(awsCfg)

	f, err := forwarder.New(&forwarder.Config{
		DestinationURI:       env.DestinationURI,
		LogPrefix:            env.LogPrefix,
		MaxFileSize:          env.MaxFileSize,
		S3Client:             s3client,
		Logger:               &logger,
		ContentTypeOverrides: env.ContentTypeOverrides,
		SourceBucketNames:    env.SourceBucketNames,
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

	entrypoint.Logger = logger
	if err := entrypoint.Register(f.Handle); err != nil {
		return fmt.Errorf("failed to register functions: %w", err)
	}
	return nil
}

func main() {
	lambda.Start(&entrypoint)
}
