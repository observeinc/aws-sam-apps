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

func (h *Handler) HandleSubscriptionRequest(ctx context.Context, subReq *SubscriptionRequest) (*Response, error) {
	var stats SubscriptionStats

	g, ctx := errgroup.WithContext(ctx)
	if h.NumWorkers > 0 {
		g.SetLimit(h.NumWorkers)
	}

	for _, logGroup := range subReq.LogGroups {
		logGroup := logGroup
		g.Go(func() error {
			if err := h.SubscribeLogGroup(ctx, logGroup, &stats); err != nil {
				return fmt.Errorf("failed to subscribe log group %q: %w", logGroup.LogGroupName, err)
			}
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("failed to subscribe log groups: %w", err)
	}

	return &Response{Subscription: &stats}, nil
}

func (h *Handler) SubscribeLogGroup(ctx context.Context, logGroup *LogGroup, stats *SubscriptionStats) error {
	logger := logr.FromContextOrDiscard(ctx).WithValues("logGroup", logGroup.LogGroupName)

	logger.V(6).Info("describing subscription filters")
	stats.Processed.Add(1)

	output, err := h.Client.DescribeSubscriptionFilters(ctx, &cloudwatchlogs.DescribeSubscriptionFiltersInput{
		LogGroupName: &logGroup.LogGroupName,
	})
	if err != nil {
		var exc *types.ResourceNotFoundException
		if errors.As(err, &exc) {
			logger.Info("log group does not exist")
			stats.Skipped.Add(1)
			return nil
		}
		return fmt.Errorf("failed to retrieve subscription filters: %w", err)
	}

	shouldExist := h.logGroupNameFilter(logGroup.LogGroupName)
	for _, action := range h.SubscriptionFilterDiff(output.SubscriptionFilters, shouldExist) {
		switch v := action.(type) {
		case *cloudwatchlogs.DeleteSubscriptionFilterInput:
			v.LogGroupName = &logGroup.LogGroupName
			logger.V(3).Info("deleting subscription filter", "filterName", aws.ToString(v.FilterName))
			if _, err := h.Client.DeleteSubscriptionFilter(ctx, v); err != nil {
				return fmt.Errorf("failed to delete subscription filter: %w", err)
			}
			stats.Deleted.Add(1)
		case *cloudwatchlogs.PutSubscriptionFilterInput:
			v.LogGroupName = &logGroup.LogGroupName
			logger.V(3).Info("updating subscription filter")
			if _, err := h.Client.PutSubscriptionFilter(ctx, v); err != nil {
				return fmt.Errorf("failed to put subscription filter: %w", err)
			}
			stats.Updated.Add(1)
		}
	}

	return nil
}

func subscriptionFilterEquals(a, b types.SubscriptionFilter) bool {
	switch {
	case aws.ToString(a.FilterName) != aws.ToString(b.FilterName):
	case aws.ToString(a.FilterPattern) != aws.ToString(b.FilterPattern):
	case aws.ToString(a.DestinationArn) != aws.ToString(b.DestinationArn):
	case aws.ToString(a.RoleArn) != aws.ToString(b.RoleArn):
	// do not match log group, since one of the arguments will be the config
	// intended for all log groups.
	default:
		return true
	}
	return false
}

// SubscriptionFilterDiff returns a list of actions to execute against
// cloudwatch API in order to converge to our intended configuration state.
func (h *Handler) SubscriptionFilterDiff(subscriptionFilters []types.SubscriptionFilter, shouldExist bool) (actions []any) {
	var exists bool

	for _, f := range subscriptionFilters {
		switch {
		case !strings.HasPrefix(aws.ToString(f.FilterName), aws.ToString(h.subscriptionFilter.FilterName)):
			// subscription filter not managed by this handler
			continue
		case subscriptionFilterEquals(h.subscriptionFilter, f):
			exists = true
		default:
			actions = append(actions, &cloudwatchlogs.DeleteSubscriptionFilterInput{
				FilterName: f.FilterName,
			})
		}
	}

	switch {
	case exists && !shouldExist:
		actions = append(actions, &cloudwatchlogs.DeleteSubscriptionFilterInput{
			FilterName: h.subscriptionFilter.FilterName,
		})
	case exists == shouldExist:
		// nothing left to do here, we already have what we need
	case aws.ToString(h.subscriptionFilter.DestinationArn) == "":
		// nothing to subscribe too, ignore
	case len(subscriptionFilters)-len(actions) < MaxSubscriptionFilterCount:
		// we can only add filter if there is space
		actions = append(actions, &cloudwatchlogs.PutSubscriptionFilterInput{
			FilterName:     h.subscriptionFilter.FilterName,
			FilterPattern:  h.subscriptionFilter.FilterPattern,
			DestinationArn: h.subscriptionFilter.DestinationArn,
			RoleArn:        h.subscriptionFilter.RoleArn,
		})
	}

	return actions
}
