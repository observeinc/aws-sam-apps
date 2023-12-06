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

	"github.com/observeinc/aws-sam-testing/logging"
)

type generator struct {
	Client           *cloudwatchlogs.Client
	LogGroupPrefix   string
	ConcurrencyLimit int
	Logger           logr.Logger
}

func (g *generator) Create(ctx context.Context, numLogGroups int) error {
	seed := time.Now().UnixNano()
	group, cctx := errgroup.WithContext(ctx)
	group.SetLimit(g.ConcurrencyLimit)

	for i := 0; i < numLogGroups; i++ {
		logGroupName := fmt.Sprintf("%s/%d-%d", g.LogGroupPrefix, seed, i)
		group.Go(func() error {
			g.Logger.Info("creating log group", "logGroupName", logGroupName)
			_, err := g.Client.CreateLogGroup(cctx, &cloudwatchlogs.CreateLogGroupInput{
				LogGroupName: aws.String(logGroupName),
			})
			return fmt.Errorf("failed to create log group: %w", err)
		})
	}

	if err := group.Wait(); err != nil {
		return fmt.Errorf("failed to generate log groups: %w", err)
	}
	return nil
}

func (g *generator) Delete(ctx context.Context) error {
	g.Logger.Info("deleting log groups")

	paginator := cloudwatchlogs.NewDescribeLogGroupsPaginator(g.Client, &cloudwatchlogs.DescribeLogGroupsInput{
		LogGroupNamePrefix: aws.String(g.LogGroupPrefix),
	})

	for paginator.HasMorePages() {
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
				_, err := g.Client.DeleteLogGroup(cctx, &cloudwatchlogs.DeleteLogGroupInput{
					LogGroupName: logGroup.LogGroupName,
				})
				return fmt.Errorf("failed to delete log group %q: %w", *logGroup.LogGroupName, err)
			})
		}

		if err := group.Wait(); err != nil {
			return fmt.Errorf("failed to delete log groups: %w", err)
		}
	}
	return nil
}

func realMain() error {
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

	ctx := context.Background()

	awsCfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to load AWS configuration: %w", err)
	}

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
	if err := realMain(); err != nil {
		panic(err)
	}
}
