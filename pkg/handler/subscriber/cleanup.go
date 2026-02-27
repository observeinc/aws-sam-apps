package subscriber

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/go-logr/logr"
	"golang.org/x/sync/errgroup"
)

// HandleCleanupRequest scans all log groups and removes subscriptions that no longer match the configured patterns.
func (h *Handler) HandleCleanupRequest(ctx context.Context, cleanupReq *CleanupRequest) (*Response, error) {
	resp := &Response{
		Subscription: new(SubscriptionStats),
	}

	logger := logr.FromContextOrDiscard(ctx)
	logger.V(3).Info("handling cleanup request", "request", cleanupReq, "dryRun", cleanupReq.DryRun)

	// Scan ALL log groups (not just matching ones)
	paginator := cloudwatchlogs.NewDescribeLogGroupsPaginator(h.Client, &cloudwatchlogs.DescribeLogGroupsInput{})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return resp, fmt.Errorf("failed to describe log groups: %w", err)
		}
		resp.Subscription.Processed.Add(int64(len(page.LogGroups)))

		// Process log groups concurrently
		g, ctx := errgroup.WithContext(ctx)
		if h.NumWorkers > 0 {
			g.SetLimit(h.NumWorkers)
		}

		for _, logGroup := range page.LogGroups {
			logGroup := logGroup
			g.Go(func() error {
				if err := h.CleanupLogGroup(ctx, aws.ToString(logGroup.LogGroupName), cleanupReq.DryRun, cleanupReq.DeleteAll, resp.Subscription); err != nil {
					return fmt.Errorf("failed to cleanup log group %q: %w", aws.ToString(logGroup.LogGroupName), err)
				}
				return nil
			})
		}

		if err := g.Wait(); err != nil {
			return resp, fmt.Errorf("failed to cleanup log groups: %w", err)
		}
	}

	logger.Info("cleanup complete", "stats", resp.Subscription)
	return resp, nil
}

// CleanupLogGroup checks if a log group has our subscription filter and removes it if it no longer matches patterns.
func (h *Handler) CleanupLogGroup(ctx context.Context, logGroupName string, dryRun bool, deleteAll bool, stats *SubscriptionStats) error {
	logger := logr.FromContextOrDiscard(ctx).WithValues("logGroup", logGroupName)

	// Get subscription filters for this log group
	output, err := h.Client.DescribeSubscriptionFilters(ctx, &cloudwatchlogs.DescribeSubscriptionFiltersInput{
		LogGroupName: aws.String(logGroupName),
	})
	if err != nil {
		logger.V(6).Info("failed to describe subscription filters", "error", err)
		return nil // Skip log groups we can't access
	}

	// Check if this log group has our subscription filter
	var hasOurFilter bool
	var filterName string
	for _, filter := range output.SubscriptionFilters {
		// Check if this filter is managed by us (matches our filter name prefix and destination)
		if aws.ToString(filter.FilterName) == aws.ToString(h.subscriptionFilter.FilterName) &&
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
		_, err = h.Client.DeleteSubscriptionFilter(ctx, &cloudwatchlogs.DeleteSubscriptionFilterInput{
			LogGroupName: aws.String(logGroupName),
			FilterName:   aws.String(filterName),
		})
		if err != nil {
			logger.Error(err, "failed to delete subscription filter")
			return fmt.Errorf("failed to delete subscription filter: %w", err)
		}
		logger.Info("subscription deleted successfully")
	}

	stats.Deleted.Add(1)
	return nil
}
