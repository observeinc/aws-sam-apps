package main

import (
	"context"
	"fmt"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/go-logr/logr"
	"github.com/sethvargo/go-envconfig"

	"github.com/observeinc/aws-sam-apps/handler"
	"github.com/observeinc/aws-sam-apps/handler/subscriber"
	"github.com/observeinc/aws-sam-apps/logging"
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
	})
	if err != nil {
		return fmt.Errorf("failed to create handler: %w", err)
	}

	entrypoint.Logger = logger
	if err := entrypoint.Register(s.HandleRequest, s.HandleSQS); err != nil {
		return fmt.Errorf("failed to register functions: %w", err)
	}
	return nil
}

func main() {
	lambda.Start(&entrypoint)
}
