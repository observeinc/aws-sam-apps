package subscriber

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"

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
