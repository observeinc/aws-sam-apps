package subscriber

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
	"github.com/go-logr/logr"
	"golang.org/x/sync/errgroup"
)

const (
	defaultCleanupMaxGroupsPerInvocation = 200
	cleanupPaginationPageSize            = 50
)

// HandleCleanupRequest scans all log groups and removes subscriptions that no longer match the configured patterns.
func (h *Handler) HandleCleanupRequest(ctx context.Context, cleanupReq *CleanupRequest) (*Response, error) {
	resp := &Response{
		Subscription: new(SubscriptionStats),
	}

	logger := logr.FromContextOrDiscard(ctx)
	logger.V(3).Info("handling cleanup request", "request", cleanupReq, "dryRun", cleanupReq.DryRun)

	maxGroups := cleanupReq.MaxGroupsPerInvocation
	if maxGroups <= 0 {
		maxGroups = defaultCleanupMaxGroupsPerInvocation
	}

	nextToken := cleanupReq.ScanToken
	remaining := maxGroups
	for remaining > 0 {
		if shouldEnqueueContinuation(ctx) {
			break
		}

		pageLimit := int32(min(remaining, cleanupPaginationPageSize))
		var output *cloudwatchlogs.DescribeLogGroupsOutput
		err := h.callCloudWatch(ctx, func() error {
			var callErr error
			output, callErr = h.Client.DescribeLogGroups(ctx, &cloudwatchlogs.DescribeLogGroupsInput{
				NextToken: nextToken,
				Limit:     aws.Int32(pageLimit),
			})
			return callErr
		})
		if err != nil {
			return resp, fmt.Errorf("failed to describe log groups: %w", err)
		}

		if len(output.LogGroups) == 0 {
			nextToken = nil
			break
		}

		if err := h.cleanupLogGroupBatch(ctx, cleanupReq, output.LogGroups, resp.Subscription); err != nil {
			return resp, err
		}

		remaining -= len(output.LogGroups)
		nextToken = output.NextToken
		if nextToken == nil {
			break
		}
	}

	if nextToken != nil {
		if h.Queue == nil {
			return resp, fmt.Errorf("cleanup continuation requires queue: %w", ErrNoQueue)
		}

		continuation := &Request{
			CleanupRequest: &CleanupRequest{
				DryRun:                 cleanupReq.DryRun,
				DeleteAll:              cleanupReq.DeleteAll,
				ScanToken:              nextToken,
				MaxGroupsPerInvocation: maxGroups,
				JobID:                  cleanupReq.JobID,
			},
		}
		if err := h.Queue.Put(ctx, continuation); err != nil {
			return resp, fmt.Errorf("failed to enqueue cleanup continuation: %w", err)
		}
		logger.Info("cleanup continuation enqueued", "jobID", cleanupReq.JobID, "nextTokenSet", true, "processed", resp.Subscription.Processed.Load())
	}

	logger.Info("cleanup complete", "stats", resp.Subscription)
	return resp, nil
}

func (h *Handler) cleanupLogGroupBatch(ctx context.Context, cleanupReq *CleanupRequest, logGroups []types.LogGroup, stats *SubscriptionStats) error {
	g, workerCtx := errgroup.WithContext(ctx)
	g.SetLimit(max(1, h.NumWorkers))

	for _, logGroup := range logGroups {
		logGroupName := aws.ToString(logGroup.LogGroupName)
		stats.Processed.Add(1)

		g.Go(func() error {
			if err := h.CleanupLogGroup(workerCtx, logGroupName, cleanupReq.DryRun, cleanupReq.DeleteAll, stats); err != nil {
				return fmt.Errorf("failed to cleanup log group %q: %w", logGroupName, err)
			}
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return fmt.Errorf("failed to cleanup log groups: %w", err)
	}
	return nil
}

// CleanupLogGroup checks if a log group has our subscription filter and removes it if it no longer matches patterns.
func (h *Handler) CleanupLogGroup(ctx context.Context, logGroupName string, dryRun bool, deleteAll bool, stats *SubscriptionStats) error {
	logger := logr.FromContextOrDiscard(ctx).WithValues("logGroup", logGroupName)

	// Get subscription filters for this log group
	var (
		output *cloudwatchlogs.DescribeSubscriptionFiltersOutput
		err    error
	)
	for attempt := 1; attempt <= cloudWatchAPIMaxAttempts; attempt++ {
		err = h.callCloudWatch(ctx, func() error {
			var callErr error
			output, callErr = h.Client.DescribeSubscriptionFilters(ctx, &cloudwatchlogs.DescribeSubscriptionFiltersInput{
				LogGroupName: aws.String(logGroupName),
			})
			return callErr
		})
		if err == nil {
			break
		}
		if !isRetryableCloudWatchError(err) || attempt == cloudWatchAPIMaxAttempts {
			break
		}
		if sleepErr := sleepWithBackoff(ctx, attempt); sleepErr != nil {
			return fmt.Errorf("describe subscription filters canceled: %w", sleepErr)
		}
	}
	if err != nil {
		var exc *types.ResourceNotFoundException
		if errors.As(err, &exc) {
			logger.V(6).Info("log group no longer exists")
			stats.Skipped.Add(1)
			return nil
		}
		return fmt.Errorf("failed to describe subscription filters: %w", err)
	}

	// Check if this log group has our subscription filter
	var hasOurFilter bool
	var filterName string
	for _, filter := range output.SubscriptionFilters {
		// Check if this filter is managed by us (name starts with our filter name prefix and destination matches)
		if strings.HasPrefix(aws.ToString(filter.FilterName), aws.ToString(h.subscriptionFilter.FilterName)) &&
			aws.ToString(filter.DestinationArn) == aws.ToString(h.subscriptionFilter.DestinationArn) {
			hasOurFilter = true
			filterName = aws.ToString(filter.FilterName)
			break
		}
	}

	if !hasOurFilter {
		// No subscription from us, nothing to clean up
		return nil
	}

	// Check if this log group should still be subscribed based on current patterns
	shouldExist := h.logGroupNameFilter(logGroupName)

	if shouldExist && !deleteAll {
		// Log group still matches patterns, keep the subscription (unless deleteAll is true)
		logger.V(6).Info("subscription still matches patterns, keeping it")
		stats.Skipped.Add(1)
		return nil
	}

	// Log group no longer matches patterns (or deleteAll is true), remove the subscription
	logger.Info("removing subscription", "reason", map[string]bool{"deleteAll": deleteAll, "noLongerMatches": !shouldExist}, "dryRun", dryRun)

	// Delete the subscription (in dry-run mode we still count it as deleted for reporting)
	if !dryRun {
		for attempt := 1; attempt <= cloudWatchAPIMaxAttempts; attempt++ {
			err = h.callCloudWatch(ctx, func() error {
				_, callErr := h.Client.DeleteSubscriptionFilter(ctx, &cloudwatchlogs.DeleteSubscriptionFilterInput{
					LogGroupName: aws.String(logGroupName),
					FilterName:   aws.String(filterName),
				})
				return callErr
			})
			if err == nil {
				break
			}
			if !isRetryableCloudWatchError(err) || attempt == cloudWatchAPIMaxAttempts {
				break
			}
			if sleepErr := sleepWithBackoff(ctx, attempt); sleepErr != nil {
				return fmt.Errorf("delete subscription filter canceled: %w", sleepErr)
			}
		}
		if err != nil {
			logger.Error(err, "failed to delete subscription filter")
			return fmt.Errorf("failed to delete subscription filter: %w", err)
		}
		logger.Info("subscription deleted successfully")
	}

	stats.Deleted.Add(1)
	return nil
}
