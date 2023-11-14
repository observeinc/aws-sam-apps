package subscriber

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"

	"github.com/observeinc/aws-sam-testing/handler"
)

var ErrNotImplemented = errors.New("not implemented")

type CloudWatchLogsClient interface {
	DescribeLogGroups(context.Context, *cloudwatchlogs.DescribeLogGroupsInput, ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.DescribeLogGroupsOutput, error)
}

type Queue interface {
	Put(context.Context, ...any) error
}

type Handler struct {
	handler.Mux

	Queue  Queue
	Client CloudWatchLogsClient
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

func (h *Handler) HandleSubscriptionRequest(_ context.Context, _ *SubscriptionRequest) (*Response, error) {
	// to be implemented
	return nil, nil
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
	}

	if cfg.Logger != nil {
		h.Logger = *cfg.Logger
	}

	if err := h.Mux.Register(h.HandleRequest); err != nil {
		return nil, fmt.Errorf("failed to register handler: %w", err)
	}

	return h, nil
}
