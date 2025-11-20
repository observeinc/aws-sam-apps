package subscriber

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"runtime"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
	"github.com/go-logr/logr"
	"golang.org/x/time/rate"

	"github.com/observeinc/aws-sam-apps/pkg/handler"
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

	Queue      Queue
	Client     CloudWatchLogsClient
	NumWorkers int
	limiter    *rate.Limiter

	subscriptionFilter types.SubscriptionFilter
	logGroupNameFilter FilterFunc
}

type FilterFunc func(string) bool

func (h *Handler) HandleRequest(ctx context.Context, req *Request) (*Response, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("failed to validate request: %w", err)
	}

	switch {
	case req.DiscoveryRequest != nil:
		return h.HandleDiscoveryRequest(ctx, req.DiscoveryRequest)
	case req.SubscriptionRequest != nil:
		return h.HandleSubscriptionRequest(ctx, req.SubscriptionRequest)
	case req.CleanupRequest != nil:
		return h.HandleCleanupRequest(ctx, req.CleanupRequest)
	default:
		return nil, ErrNotImplemented
	}
}

func (h *Handler) HandleSQS(ctx context.Context, request events.SQSEvent) (response events.SQSEventResponse, err error) {
	logger := logr.FromContextOrDiscard(ctx)
	for _, record := range request.Records {
		var req Request
		var err error

		if err = json.Unmarshal([]byte(record.Body), &req); err == nil {
			_, err = h.HandleRequest(ctx, &req)
		}

		if err != nil {
			// SQS record will be under 256KB, should be ok to log
			logger.Error(err, "failed to process request", "body", record.Body)
			response.BatchItemFailures = append(response.BatchItemFailures, events.SQSBatchItemFailure{
				ItemIdentifier: record.MessageId,
			})
		}
	}
	return response, nil
}

func New(cfg *Config) (*Handler, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	h := &Handler{
		Client:     cfg.CloudWatchLogsClient,
		Queue:      cfg.Queue,
		NumWorkers: cfg.NumWorkers,
		subscriptionFilter: types.SubscriptionFilter{
			FilterName:     aws.String(cfg.FilterName),
			FilterPattern:  aws.String(cfg.FilterPattern),
			DestinationArn: aws.String(cfg.DestinationARN),
			RoleArn:        cfg.RoleARN,
		},
		logGroupNameFilter: cfg.LogGroupFilter(),
	}

	if h.NumWorkers <= 0 {
		h.NumWorkers = runtime.NumCPU()
	}

	rps := cfg.CloudWatchAPIRateLimit
	if rps <= 0 {
		rps = 8
	}
	burst := cfg.CloudWatchAPIBurst
	if burst <= 0 {
		burst = int(math.Ceil(rps * 2))
		if burst < 1 {
			burst = 1
		}
	}
	h.limiter = rate.NewLimiter(rate.Limit(rps), burst)

	return h, nil
}

func (h *Handler) callCloudWatch(ctx context.Context, fn func() error) error {
	if h.limiter != nil {
		if err := h.limiter.Wait(ctx); err != nil {
			return err
		}
	}
	return fn()
}


