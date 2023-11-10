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

	"github.com/observeinc/aws-sam-testing/handler"
)

var (
	MaxSubscriptionFilterCount = 2
	ErrNotImplemented          = errors.New("not implemented")
)

type CloudWatchLogsClient interface {
	DescribeLogGroups(context.Context, *cloudwatchlogs.DescribeLogGroupsInput, ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.DescribeLogGroupsOutput, error)
	DescribeSubscriptionFilters(context.Context, *cloudwatchlogs.DescribeSubscriptionFiltersInput, ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.DescribeSubscriptionFiltersOutput, error)
	PutSubscriptionFilter(context.Context, *cloudwatchlogs.PutSubscriptionFilterInput, ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.PutSubscriptionFilterOutput, error)
	DeleteSubscriptionFilter(context.Context, *cloudwatchlogs.DeleteSubscriptionFilterInput, ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.DeleteSubscriptionFilterOutput, error)
}

type Queue interface {
	Put(context.Context, ...any) error
}

type Handler struct {
	handler.Mux

	Queue  Queue
	Client CloudWatchLogsClient

	subscriptionFilter types.SubscriptionFilter
}

func (h *Handler) HandleDiscoveryRequest(ctx context.Context, discoveryReq *DiscoveryRequest) (*Response, error) {
	var discoveryResp DiscoveryResponse

	for _, input := range discoveryReq.ToDescribeLogInputs() {
		paginator := cloudwatchlogs.NewDescribeLogGroupsPaginator(h.Client, input)

		for paginator.HasMorePages() {
			page, err := paginator.NextPage(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to describe log groups: %w", err)
			}
			discoveryResp.RequestCount++
			discoveryResp.LogGroupCount += len(page.LogGroups)
		}
	}

	return &Response{DiscoveryResponse: &discoveryResp}, nil
}

func (h *Handler) HandleSubscriptionRequest(ctx context.Context, subReq *SubscriptionRequest) (*Response, error) {
	for _, logGroup := range subReq.LogGroups {
		if err := h.SubscribeLogGroup(ctx, logGroup); err != nil {
			return nil, fmt.Errorf("failed to subscribe log group: %w", err)
		}
	}

	return nil, nil
}

func (h *Handler) SubscribeLogGroup(ctx context.Context, logGroup *LogGroup) error {
	logger := logr.FromContextOrDiscard(ctx).WithValues("logGroup", logGroup.LogGroupName)

	logger.V(6).Info("describing subscription filters")
	output, err := h.Client.DescribeSubscriptionFilters(ctx, &cloudwatchlogs.DescribeSubscriptionFiltersInput{
		LogGroupName: &logGroup.LogGroupName,
	})
	if err != nil {
		var exc *types.ResourceNotFoundException
		if errors.As(err, &exc) {
			logger.Info("skipping log group")
			return nil
		}
		return fmt.Errorf("failed to retrieve subscription filters: %w", err)
	}

	for _, action := range h.SubscriptionFilterDiff(output.SubscriptionFilters) {
		switch v := action.(type) {
		case *cloudwatchlogs.DeleteSubscriptionFilterInput:
			v.LogGroupName = &logGroup.LogGroupName
			logger.V(3).Info("deleting subscription filter", "filterName", *v.FilterName)
			if _, err := h.Client.DeleteSubscriptionFilter(ctx, v); err != nil {
				return fmt.Errorf("failed to delete subscription filter: %w", err)
			}
		case *cloudwatchlogs.PutSubscriptionFilterInput:
			v.LogGroupName = &logGroup.LogGroupName
			logger.V(3).Info("updating subscription filter")
			if _, err := h.Client.PutSubscriptionFilter(ctx, v); err != nil {
				return fmt.Errorf("failed to put subscription filter: %w", err)
			}
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
func (h *Handler) SubscriptionFilterDiff(subscriptionFilters []types.SubscriptionFilter) (actions []any) {
	var (
		deleted, updated int
		deleteOnly       = aws.ToString(h.subscriptionFilter.DestinationArn) == ""
	)

	for _, f := range subscriptionFilters {
		if !strings.HasPrefix(aws.ToString(f.FilterName), aws.ToString(h.subscriptionFilter.FilterName)) {
			// subscription filter not managed by this handler
			continue
		}
		if deleteOnly || aws.ToString(h.subscriptionFilter.FilterName) != aws.ToString(f.FilterName) {
			deleted++
			actions = append(actions, &cloudwatchlogs.DeleteSubscriptionFilterInput{
				FilterName: f.FilterName,
			})
		} else if !subscriptionFilterEquals(h.subscriptionFilter, f) {
			updated++
			actions = append(actions, &cloudwatchlogs.PutSubscriptionFilterInput{
				FilterName:     h.subscriptionFilter.FilterName,
				FilterPattern:  h.subscriptionFilter.FilterPattern,
				DestinationArn: h.subscriptionFilter.DestinationArn,
				RoleArn:        h.subscriptionFilter.LogGroupName,
			})
		}
	}

	if !deleteOnly && updated == 0 && len(subscriptionFilters)-deleted < MaxSubscriptionFilterCount {
		actions = append(actions, &cloudwatchlogs.PutSubscriptionFilterInput{
			FilterName:     h.subscriptionFilter.FilterName,
			FilterPattern:  h.subscriptionFilter.FilterPattern,
			DestinationArn: h.subscriptionFilter.DestinationArn,
			RoleArn:        h.subscriptionFilter.LogGroupName,
		})
	}

	return actions
}

func (h *Handler) HandleRequest(ctx context.Context, req *Request) (*Response, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("failed to validate request: %w", err)
	}

	switch {
	case req.DiscoveryRequest != nil:
		return h.HandleDiscoveryRequest(ctx, req.DiscoveryRequest)
	case req.SubscriptionRequest != nil:
		return h.HandleSubscriptionRequest(ctx, req.SubscriptionRequest)
	default:
		return nil, ErrNotImplemented
	}
}

func New(cfg *Config) (*Handler, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	h := &Handler{
		Client: cfg.CloudWatchLogsClient,
		Queue:  cfg.Queue,
		subscriptionFilter: types.SubscriptionFilter{
			FilterName:     aws.String(cfg.FilterName),
			FilterPattern:  aws.String(cfg.FilterPattern),
			DestinationArn: aws.String(cfg.DestinationARN),
			RoleArn:        aws.String(cfg.RoleARN),
		},
	}

	if cfg.Logger != nil {
		h.Logger = *cfg.Logger
	}

	if err := h.Mux.Register(h.HandleRequest); err != nil {
		return nil, fmt.Errorf("failed to register handler: %w", err)
	}

	return h, nil
}
