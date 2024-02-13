package subscriber

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/go-logr/logr"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var ErrNoQueue = errors.New("no queue defined")

func (h *Handler) HandleDiscoveryRequest(ctx context.Context, discoveryReq *DiscoveryRequest) (*Response, error) {
	ctx, span := h.Tracer.Start(ctx, "HandleDiscoveryRequest", trace.WithAttributes(attribute.String("key1", "value1")))
	var err error
	defer func() {
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()

	resp := &Response{
		Discovery: new(DiscoveryStats),
	}

	logger := logr.FromContextOrDiscard(ctx)
	logger.V(3).Info("handling discovery request", "request", discoveryReq)

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
