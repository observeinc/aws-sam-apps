package main

import (
	"context"
	"flag"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/go-logr/logr"
	"golang.org/x/sync/errgroup"

	"go.opentelemetry.io/contrib/exporters/autoexport"
	"go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"

	"github.com/observeinc/aws-sam-apps/logging"
)

const (
	instrumentationName    = "github.com/observeinc/aws-sam-apps/cmd/loggroupgenerator"
	instrumentationVersion = "0.1.0"
)

var tracer = otel.GetTracerProvider().Tracer(
	instrumentationName,
	trace.WithInstrumentationVersion(instrumentationVersion),
)

type generator struct {
	Client           *cloudwatchlogs.Client
	LogGroupPrefix   string
	ConcurrencyLimit int
	Logger           logr.Logger
}

func (g *generator) Create(ctx context.Context, numLogGroups int) error {
	ctx, span := tracer.Start(ctx, "create")
	defer span.End()

	seed := time.Now().UnixNano()
	group, cctx := errgroup.WithContext(ctx)
	group.SetLimit(g.ConcurrencyLimit)

	for i := 0; i < numLogGroups; i++ {
		logGroupName := fmt.Sprintf("%s/%d-%d", g.LogGroupPrefix, seed, i)
		group.Go(func() error {
			g.Logger.Info("creating log group", "logGroupName", logGroupName)
			if _, err := g.Client.CreateLogGroup(cctx, &cloudwatchlogs.CreateLogGroupInput{
				LogGroupName: aws.String(logGroupName),
			}); err != nil {
				return fmt.Errorf("failed to create log group: %w", err)
			}
			return nil
		})
	}

	if err := group.Wait(); err != nil {
		return fmt.Errorf("failed to generate log groups: %w", err)
	}
	return nil
}

func (g *generator) Delete(ctx context.Context) error {
	ctx, span := tracer.Start(ctx, "delete")
	defer span.End()

	g.Logger.Info("deleting log groups")

	paginator := cloudwatchlogs.NewDescribeLogGroupsPaginator(g.Client, &cloudwatchlogs.DescribeLogGroupsInput{
		LogGroupNamePrefix: aws.String(g.LogGroupPrefix),
	})

	for paginator.HasMorePages() {
		if err := g.processPageDelete(ctx, paginator); err != nil {
			return fmt.Errorf("failed to process page: %w", err)
		}
	}
	return nil
}

func (g *generator) processPageDelete(ctx context.Context, paginator *cloudwatchlogs.DescribeLogGroupsPaginator) error {
	ctx, span := tracer.Start(ctx, "processPage")
	defer span.End()

	page, err := paginator.NextPage(ctx)
	if err != nil {
		return fmt.Errorf("failed to get page: %w", err)
	}

	group, cctx := errgroup.WithContext(ctx)
	group.SetLimit(g.ConcurrencyLimit)
	for _, logGroup := range page.LogGroups {
		logGroup := logGroup
		group.Go(func() error {
			g.Logger.Info("deleting", "name", logGroup.LogGroupName)
			if _, err := g.Client.DeleteLogGroup(cctx, &cloudwatchlogs.DeleteLogGroupInput{
				LogGroupName: logGroup.LogGroupName,
			}); err != nil {
				return fmt.Errorf("failed to delete log group %q: %w", *logGroup.LogGroupName, err)
			}
			return nil
		})
	}

	if err := group.Wait(); err != nil {
		return fmt.Errorf("failed to delete log groups: %w", err)
	}
	return nil
}

func realMain(ctx context.Context) error {
	var (
		verbosity        = flag.Int("verbosity", 9, "Log verbosity")
		numLogGroups     = flag.Int("num-log-groups", 0, "Number of log groups to generate")
		concurrencyLimit = flag.Int("concurrency-limit", -1, "Maximum number of concurrent API calls. A negative value indicates no limit.")
		deleteExisting   = flag.Bool("delete-existing", false, "Delete log groups matching provided prefix.")
		logGroupPrefix   = flag.String("log-group-prefix", "/generated", "Prefix used for generated log groups")
	)
	flag.Parse()

	logger := logging.New(&logging.Config{
		Verbosity: *verbosity,
	})

	awsCfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to load AWS configuration: %w", err)
	}

	otelaws.AppendMiddlewares(&awsCfg.APIOptions)

	generator := generator{
		Client:           cloudwatchlogs.NewFromConfig(awsCfg),
		LogGroupPrefix:   *logGroupPrefix,
		ConcurrencyLimit: *concurrencyLimit,
		Logger:           logger,
	}

	if *deleteExisting {
		if err := generator.Delete(ctx); err != nil {
			return fmt.Errorf("failed to delete log groups: %w", err)
		}
	}

	if *numLogGroups > 0 {
		if err := generator.Create(ctx, *numLogGroups); err != nil {
			return fmt.Errorf("failed to create log groups: %w", err)
		}
	}

	return nil
}

func main() {
	ctx := context.Background()
	exporter, err := autoexport.NewSpanExporter(ctx)
	if err != nil {
		panic(err)
	}

	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(resource.Default()),
	)
	otel.SetTracerProvider(tracerProvider)

	defer func() {
		if err := tracerProvider.Shutdown(ctx); err != nil {
			panic(err)
		}
	}()

	ctx, span := tracer.Start(context.Background(), "invocation", trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	if err := realMain(ctx); err != nil {
		span.SetStatus(codes.Error, "loggroupgenerator failed")
		span.RecordError(err)
		panic(err)
	}
}
