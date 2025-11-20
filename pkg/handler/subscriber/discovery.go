package subscriber

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/go-logr/logr"
)

var ErrNoQueue = errors.New("no queue defined")

const (
	defaultDiscoveryMaxGroupsPerInvocation = 200
	discoveryPaginationPageSize            = 50
	continuationSafetyWindow               = 20 * time.Second
)

func (h *Handler) HandleDiscoveryRequest(ctx context.Context, discoveryReq *DiscoveryRequest) (*Response, error) {
	resp := &Response{
		Discovery: new(DiscoveryStats),
	}

	logger := logr.FromContextOrDiscard(ctx)
	logger.V(3).Info("handling discovery request", "request", discoveryReq, "fullyPrune", discoveryReq.FullyPrune)

	// Prefer request-scoped filter criteria when provided so that queued discovery
	// work is evaluated against the same pattern set that triggered it.
	if len(discoveryReq.LogGroupNamePatterns) > 0 || len(discoveryReq.LogGroupNamePrefixes) > 0 || len(discoveryReq.ExcludeLogGroupNamePatterns) > 0 {
		originalFilter := h.logGroupNameFilter
		h.logGroupNameFilter = BuildLogGroupFilter(
			ptrSliceToStrSlice(discoveryReq.LogGroupNamePatterns),
			ptrSliceToStrSlice(discoveryReq.LogGroupNamePrefixes),
			ptrSliceToStrSlice(discoveryReq.ExcludeLogGroupNamePatterns),
		)
		defer func() {
			h.logGroupNameFilter = originalFilter
		}()
	}

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

	maxGroups := discoveryReq.MaxGroupsPerInvocation
	if maxGroups <= 0 {
		maxGroups = defaultDiscoveryMaxGroupsPerInvocation
	}

	// Build the scan plan. FullyPrune scans the entire namespace with one input.
	inputs := discoveryReq.ToDescribeLogInputs()
	if discoveryReq.FullyPrune {
		logger.Info("fully pruning: scanning all log groups")
		inputs = []*cloudwatchlogs.DescribeLogGroupsInput{{}}
	}

	if len(inputs) == 0 {
		return resp, nil
	}

	if discoveryReq.ScanInputIndex < 0 || discoveryReq.ScanInputIndex >= len(inputs) {
		return resp, fmt.Errorf("invalid scanInputIndex: %d", discoveryReq.ScanInputIndex)
	}

	continuationInputIndex := -1
	var continuationToken *string
	remaining := maxGroups
	for inputIdx := discoveryReq.ScanInputIndex; inputIdx < len(inputs) && remaining > 0; inputIdx++ {
		baseInput := inputs[inputIdx]
		nextToken := aws.ToString(discoveryReq.ScanToken)
		if inputIdx != discoveryReq.ScanInputIndex {
			nextToken = ""
		}

		for remaining > 0 {
			if shouldEnqueueContinuation(ctx) {
				continuationInputIndex = inputIdx
				if nextToken != "" {
					continuationToken = aws.String(nextToken)
				}
				break
			}

			pageLimit := int32(min(remaining, discoveryPaginationPageSize))
			callInput := *baseInput
			if nextToken != "" {
				callInput.NextToken = aws.String(nextToken)
			}
			if callInput.Limit != nil {
				pageLimit = min(pageLimit, *callInput.Limit)
			}
			callInput.Limit = aws.Int32(pageLimit)

			var page *cloudwatchlogs.DescribeLogGroupsOutput
			err := h.callCloudWatch(ctx, func() error {
				var callErr error
				page, callErr = h.Client.DescribeLogGroups(ctx, &callInput)
				return callErr
			})
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

			remaining -= len(page.LogGroups)
			if page.NextToken == nil {
				nextToken = ""
				break
			}
			nextToken = aws.ToString(page.NextToken)
		}

		if continuationInputIndex >= 0 {
			break
		}

		if nextToken != "" {
			continuationInputIndex = inputIdx
			continuationToken = aws.String(nextToken)
			break
		}
	}

	if continuationInputIndex >= 0 {
		if h.Queue == nil {
			return resp, fmt.Errorf("discovery continuation requires queue: %w", ErrNoQueue)
		}

		continuation := &Request{
			DiscoveryRequest: &DiscoveryRequest{
				LogGroupNamePatterns:        discoveryReq.LogGroupNamePatterns,
				LogGroupNamePrefixes:        discoveryReq.LogGroupNamePrefixes,
				ExcludeLogGroupNamePatterns: discoveryReq.ExcludeLogGroupNamePatterns,
				Limit:                       discoveryReq.Limit,
				Inline:                      discoveryReq.Inline,
				FullyPrune:                  discoveryReq.FullyPrune,
				ScanToken:                   continuationToken,
				ScanInputIndex:              continuationInputIndex,
				MaxGroupsPerInvocation:      maxGroups,
				JobID:                       discoveryReq.JobID,
			},
		}
		if err := h.Queue.Put(ctx, continuation); err != nil {
			return resp, fmt.Errorf("failed to enqueue discovery continuation: %w", err)
		}
		logger.Info("discovery continuation enqueued", "jobID", discoveryReq.JobID, "scanInputIndex", continuationInputIndex, "nextTokenSet", continuationToken != nil, "processed", resp.Discovery.LogGroupCount.Load())
	}

	return resp, nil
}

func shouldEnqueueContinuation(ctx context.Context) bool {
	deadline, ok := ctx.Deadline()
	if !ok {
		return false
	}
	return time.Until(deadline) <= continuationSafetyWindow
}
