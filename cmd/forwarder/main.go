package main

import (
	"context"
	"fmt"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/go-logr/logr"
	"github.com/sethvargo/go-envconfig"

	"github.com/observeinc/aws-sam-apps/handler/forwarder"
	"github.com/observeinc/aws-sam-apps/handler/forwarder/override"
	"github.com/observeinc/aws-sam-apps/logging"
)

var env struct {
	DestinationURI       string           `env:"DESTINATION_URI,required"`
	LogPrefix            string           `env:"LOG_PREFIX,default=forwarder/"`
	Verbosity            int              `env:"VERBOSITY,default=1"`
	MaxFileSize          int64            `env:"MAX_FILE_SIZE"`
	ContentTypeOverrides []*override.Rule `env:"CONTENT_TYPE_OVERRIDES"`
	PresetOverrides      []string         `env:"PRESET_OVERRIDES,default=aws/v1"`
	SourceBucketNames    []string         `env:"SOURCE_BUCKET_NAMES"`
}

var (
	logger  logr.Logger
	handler *forwarder.Handler
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

	handler, err = forwarder.New(&forwarder.Config{
		DestinationURI:    env.DestinationURI,
		LogPrefix:         env.LogPrefix,
		MaxFileSize:       env.MaxFileSize,
		S3Client:          s3client,
		Logger:            &logger,
		Override:          append(override.Sets{customOverrides}, presets...),
		SourceBucketNames: env.SourceBucketNames,
	})
	if err != nil {
		return fmt.Errorf("failed to create handler: %w", err)
	}

	region, err := handler.GetDestinationRegion(ctx, s3client)
	if err != nil {
		return fmt.Errorf("failed to get destination region: %w", err)
	}

	if awsCfg.Region != region {
		logger.V(4).Info("modifying s3 client region", "region", region)
		regionCfg := awsCfg.Copy()
		regionCfg.Region = region
		handler.S3Client = s3.NewFromConfig(regionCfg)
	}

	return nil
}

func main() {
	lambda.Start(handler)
}
