package subscriber

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/go-logr/logr"

	"github.com/observeinc/aws-sam-testing/handler"
)

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

type SyncRequest struct {
	*SyncConfig `json:"sync"`
}

type SyncConfig struct {
	Subscription *cloudwatchlogs.PutSubscriptionFilterInput `json:"subscription,omitempty"`
	Limit        *int32                                     `json:"limit,omitempty"`
}

type task struct {
	PutSubscriptionFilterInput *cloudwatchlogs.PutSubscriptionFilterInput `json:"subscription,omitempty"`
	DescribeLogGroupsOutput    *cloudwatchlogs.DescribeLogGroupsOutput    `json:"logGroups"`
}

type SyncResponse struct {
	LogGroupCount int `json:"logGroupCount"`
	PageCount     int `json:"pageCount"`
}

func (h *Handler) HandleSync(ctx context.Context, request SyncRequest) (*SyncResponse, error) {
	logger := logr.FromContextOrDiscard(ctx)

	paginator := cloudwatchlogs.NewDescribeLogGroupsPaginator(h.Client, &cloudwatchlogs.DescribeLogGroupsInput{
		Limit: request.Limit,
	})

	var response SyncResponse

	for paginator.HasMorePages() {
		output, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to describe log groups: %w", err)
		}

		response.PageCount++
		response.LogGroupCount += len(output.LogGroups)

		logger.V(6).Info("queueing page")

		if err := h.Queue.Put(ctx, &task{
			PutSubscriptionFilterInput: request.Subscription,
			DescribeLogGroupsOutput:    output,
		}); err != nil {
			return nil, fmt.Errorf("failed to queue log groups: %w", err)
		}
	}

	return &response, nil
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

	if err := h.Mux.Register(h.HandleSync); err != nil {
		return nil, fmt.Errorf("failed to register handler: %w", err)
	}

	return h, nil
}
