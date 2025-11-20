package subscriber

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/go-logr/logr"
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

	// If FullyPrune is enabled, scan ALL log groups and let SubscribeLogGroup handle
	// both subscribing matching log groups and removing subscriptions from non-matching ones.
	// This is used during stack updates when patterns change.
	if discoveryReq.FullyPrune {
		logger.Info("fully pruning: scanning all log groups")

		paginator := cloudwatchlogs.NewDescribeLogGroupsPaginator(h.Client, &cloudwatchlogs.DescribeLogGroupsInput{})

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

		logger.Info("fully prune complete", "stats", resp.Discovery.Subscription)
		return resp, nil
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
