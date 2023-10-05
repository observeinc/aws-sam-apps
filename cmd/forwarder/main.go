package main

import (
	"context"
	"fmt"
	stdlog "log"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/go-logr/logr"
	"github.com/go-logr/stdr"
	"github.com/sethvargo/go-envconfig"

	"github.com/observeinc/aws-sam-testing/handlers/forwarder"
)

var env struct {
	DestinationURI string `env:"DESTINATION_URI,required"`
	LogPrefix      string `env:"LOG_PREFIX,default=forwarder/"`
	Verbosity      int    `env:"VERBOSITY,default=1"`
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

	stdr.SetVerbosity(env.Verbosity)
	logger = stdr.NewWithOptions(stdlog.New(os.Stderr, "", stdlog.LstdFlags), stdr.Options{LogCaller: stdr.All})
	logger.V(6).Info("initialized", "config", env)

	awsCfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to load AWS configuration: %w", err)
	}

	s3client := s3.NewFromConfig(awsCfg)

	handler, err = forwarder.New(&forwarder.Config{
		DestinationURI: env.DestinationURI,
		LogPrefix:      env.LogPrefix,
		S3Client:       s3client,
		Logger:         &logger,
	})
	if err != nil {
		return fmt.Errorf("failed to create handler: %w", err)
	}

	region, err := handler.GetDestinationRegion(ctx, s3client)
	if err != nil {
		return fmt.Errorf("failed to get destination region: %w", err)
	}

	if awsCfg.Region != region {
		logger.V(6).Info("modifying s3 client region", "region", region)
		regionCfg := awsCfg.Copy()
		regionCfg.Region = region
		handler.S3Client = s3.NewFromConfig(regionCfg)
	}

	return nil
}

func main() {
	lambda.Start(handler.Handle)
}
