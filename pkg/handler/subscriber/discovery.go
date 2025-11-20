package subscriber

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/go-logr/logr"
	"golang.org/x/sync/errgroup"
)

var ErrNoQueue = errors.New("no queue defined")

func (h *Handler) HandleDiscoveryRequest(ctx context.Context, discoveryReq *DiscoveryRequest) (*Response, error) {
	resp := &Response{
		Discovery: new(DiscoveryStats),
	}

	logger := logr.FromContextOrDiscard(ctx)
	logger.V(3).Info("handling discovery request", "request", discoveryReq, "fullyPrune", discoveryReq.FullyPrune)

	var inline bool
	if discoveryReq.Inline == nil {
		inline = h.Queue == nil
	} else {
		inline = *discoveryReq.Inline
	}

	if !inline && h.Queue == nil {
		return resp, fmt.Errorf("cannot fan out: %w", ErrNoQueue)
	} else if inline {
		resp.Discovery.Subscription = new(SubscriptionStats)
	}

	// If FullyPrune is enabled, scan ALL log groups to find and remove stale subscriptions
	// This is used during stack updates when patterns change
	if discoveryReq.FullyPrune {
		logger.Info("fully pruning stale subscriptions")
		resp.Discovery.Cleanup = new(CleanupStats)

		if err := h.pruneStaleSubscriptions(ctx, resp.Discovery.Cleanup); err != nil {
			return resp, fmt.Errorf("failed to prune stale subscriptions: %w", err)
		}
		logger.Info("prune complete", "stats", resp.Discovery.Cleanup)
	}

	// Discover and subscribe log groups matching the patterns
	for _, input := range discoveryReq.ToDescribeLogInputs() {
		paginator := cloudwatchlogs.NewDescribeLogGroupsPaginator(h.Client, input)

		for paginator.HasMorePages() {
			page, err := paginator.NextPage(ctx)
			if err != nil {
				return resp, fmt.Errorf("failed to describe log groups: %w", err)
			}
			resp.Discovery.RequestCount.Add(1)
			resp.Discovery.LogGroupCount.Add(int64(len(page.LogGroups)))

			subscriptionRequest := NewSubscriptionRequestFromLogGroupsOutput(page)

			if inline {
				s, err := h.HandleSubscriptionRequest(ctx, subscriptionRequest)
				if err != nil {
					return resp, fmt.Errorf("failed to handle subscription request: %w", err)
				}
				resp.Discovery.Subscription.Add(s.Subscription)
			} else if err := h.Queue.Put(ctx, &Request{SubscriptionRequest: subscriptionRequest}); err != nil {
				return resp, fmt.Errorf("failed to write to queue: %w", err)
			}
		}
	}

	return resp, nil
}

// pruneStaleSubscriptions scans ALL log groups and removes subscriptions managed by us
// that no longer match the current filter patterns.
func (h *Handler) pruneStaleSubscriptions(ctx context.Context, stats *CleanupStats) error {
	logger := logr.FromContextOrDiscard(ctx)

	// Scan ALL log groups (not just matching ones)
	paginator := cloudwatchlogs.NewDescribeLogGroupsPaginator(h.Client, &cloudwatchlogs.DescribeLogGroupsInput{})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return fmt.Errorf("failed to describe log groups: %w", err)
		}
		stats.LogGroupsScanned.Add(int64(len(page.LogGroups)))

		// Process log groups concurrently
		g, ctx := errgroup.WithContext(ctx)
		if h.NumWorkers > 0 {
			g.SetLimit(h.NumWorkers)
		}

		for _, logGroup := range page.LogGroups {
			g.Go(func() error {
				return h.pruneLogGroupSubscription(ctx, aws.ToString(logGroup.LogGroupName), stats)
			})
		}

		if err := g.Wait(); err != nil {
			return err
		}
	}

	logger.V(3).Info("prune scan complete", "logGroupsScanned", stats.LogGroupsScanned.Load())
	return nil
}

// pruneLogGroupSubscription checks if a log group has our subscription filter and removes it
// if it no longer matches the current patterns.
func (h *Handler) pruneLogGroupSubscription(ctx context.Context, logGroupName string, stats *CleanupStats) error {
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
		// Check if this filter is managed by us (matches our filter name and destination)
		if aws.ToString(filter.FilterName) == aws.ToString(h.subscriptionFilter.FilterName) &&
			aws.ToString(filter.DestinationArn) == aws.ToString(h.subscriptionFilter.DestinationArn) {
			hasOurFilter = true
			filterName = aws.ToString(filter.FilterName)
			break
		}
	}

	if !hasOurFilter {
		// No subscription from us, nothing to prune
		return nil
	}

	stats.SubscriptionsFound.Add(1)

	// Check if this log group should still be subscribed based on current patterns
	shouldExist := h.logGroupNameFilter(logGroupName)

	if shouldExist {
		// Log group still matches patterns, keep the subscription
		logger.V(6).Info("subscription still matches patterns, keeping it")
		stats.SubscriptionsKept.Add(1)
		return nil
	}

	// Log group no longer matches patterns, remove the subscription
	logger.Info("removing stale subscription", "reason", "noLongerMatchesPatterns")

	_, err = h.Client.DeleteSubscriptionFilter(ctx, &cloudwatchlogs.DeleteSubscriptionFilterInput{
		LogGroupName: aws.String(logGroupName),
		FilterName:   aws.String(filterName),
	})
	if err != nil {
		logger.Error(err, "failed to delete subscription filter")
		return fmt.Errorf("failed to delete subscription filter: %w", err)
	}

	stats.SubscriptionsDeleted.Add(1)
	logger.Info("stale subscription deleted successfully")
	return nil
}
