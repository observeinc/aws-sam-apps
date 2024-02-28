package main

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/go-logr/logr"
	"github.com/sethvargo/go-envconfig"
	"go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws"

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
	ServiceName          string   `env:"OTEL_SERVICE_NAME,default=subscriber"`
	AWSMaxAttempts       string   `env:"AWS_MAX_ATTEMPTS,default=7"`
	AWSRetryMode         string   `env:"AWS_RETRY_MODE,default=adaptive"`
}

var (
	logger     logr.Logger
	entrypoint handler.Mux
	options    []lambda.Option
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

	// If user has not provided an override, we must reapply these environment
	// variables for our defaults to take hold.
	// Sadly, this approach is simpler than attempting to adjust the
	// aws.Config struct ourselves.
	os.Setenv("AWS_MAX_ATTEMPTS", env.AWSMaxAttempts)
	os.Setenv("AWS_RETRY_MODE", env.AWSRetryMode)

	awsCfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to load AWS configuration: %w", err)
	}

	tracer, tracerShutdownFn := subscriber.InitTracing(ctx, env.ServiceName)
	if tracer == nil {
		err := tracerShutdownFn(ctx)
		return fmt.Errorf("failed to initialize tracing: %w", err)
	}
	options = append(options, lambda.WithEnableSIGTERM(func() {
		logger.V(4).Info("SIGTERM received, running shutdown")
		err := tracerShutdownFn(ctx)
		if err != nil {
			logger.V(4).Error(err, "tracer shutdown failed")
		}
		logger.V(4).Info("shutdown done running")
	}))
	otelaws.AppendMiddlewares(&awsCfg.APIOptions)

	queue, err := subscriber.NewQueue(sqs.NewFromConfig(awsCfg), env.QueueURL)
	iq := subscriber.InstrumentQueue(*queue)
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
		Queue:                &iq,
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
	lambda.StartWithOptions(&entrypoint, options...)
}
