package subscriber_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"

	"github.com/observeinc/aws-sam-apps/pkg/handler/subscriber"
	"github.com/observeinc/aws-sam-apps/pkg/testing/awstest"
)

type queueRecorder struct {
	items []any
}

func (q *queueRecorder) Put(_ context.Context, items ...any) error {
	q.items = append(q.items, items...)
	return nil
}

func TestHandleCleanupRequestEnqueuesContinuation(t *testing.T) {
	t.Parallel()

	const total = 120
	logGroups := make([]types.LogGroup, 0, total)
	filters := make([]types.SubscriptionFilter, 0, total)
	for i := range total {
		name := fmt.Sprintf("/aws/lambda/test-%03d", i)
		logGroups = append(logGroups, types.LogGroup{
			LogGroupName: aws.String(name),
		})
		filters = append(filters, types.SubscriptionFilter{
			FilterName:     aws.String("test"),
			LogGroupName:   aws.String(name),
			DestinationArn: aws.String("arn:aws:lambda:us-west-2:123456789012:function:example"),
		})
	}

	queue := &queueRecorder{}
	client := &awstest.CloudWatchLogsClient{
		LogGroups:           logGroups,
		SubscriptionFilters: filters,
	}

	h, err := subscriber.New(&subscriber.Config{
		CloudWatchLogsClient: client,
		Queue:                queue,
		FilterName:           "test",
		DestinationARN:       "arn:aws:lambda:us-west-2:123456789012:function:example",
		LogGroupNamePrefixes: []string{"*"},
		NumWorkers:           4,
	})
	if err != nil {
		t.Fatal(err)
	}

	resp, err := h.HandleCleanupRequest(context.Background(), &subscriber.CleanupRequest{
		DeleteAll:              true,
		MaxGroupsPerInvocation: 50,
		JobID:                  "job-1",
	})
	if err != nil {
		t.Fatal(err)
	}

	if got := resp.Subscription.Processed.Load(); got != 50 {
		t.Fatalf("processed=%d want=50", got)
	}
	if got := len(queue.items); got != 1 {
		t.Fatalf("queued messages=%d want=1", got)
	}

	req, ok := queue.items[0].(*subscriber.Request)
	if !ok || req == nil || req.CleanupRequest == nil {
		t.Fatalf("unexpected continuation payload type: %#v", queue.items[0])
	}
	if req.CleanupRequest.ScanToken == nil || *req.CleanupRequest.ScanToken == "" {
		t.Fatalf("continuation missing scanToken: %#v", req.CleanupRequest)
	}
	if req.CleanupRequest.MaxGroupsPerInvocation != 50 {
		t.Fatalf("continuation maxGroups=%d want=50", req.CleanupRequest.MaxGroupsPerInvocation)
	}
}

func TestHandleCleanupRequestCompletesWithoutContinuation(t *testing.T) {
	t.Parallel()

	const total = 40
	logGroups := make([]types.LogGroup, 0, total)
	filters := make([]types.SubscriptionFilter, 0, total)
	for i := range total {
		name := fmt.Sprintf("/aws/ecs/test-%03d", i)
		logGroups = append(logGroups, types.LogGroup{
			LogGroupName: aws.String(name),
		})
		filters = append(filters, types.SubscriptionFilter{
			FilterName:     aws.String("test"),
			LogGroupName:   aws.String(name),
			DestinationArn: aws.String("arn:aws:lambda:us-west-2:123456789012:function:example"),
		})
	}

	queue := &queueRecorder{}
	client := &awstest.CloudWatchLogsClient{
		LogGroups:           logGroups,
		SubscriptionFilters: filters,
		DeleteSubscriptionFilterFunc: func(context.Context, *cloudwatchlogs.DeleteSubscriptionFilterInput, ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.DeleteSubscriptionFilterOutput, error) {
			return &cloudwatchlogs.DeleteSubscriptionFilterOutput{}, nil
		},
	}

	h, err := subscriber.New(&subscriber.Config{
		CloudWatchLogsClient: client,
		Queue:                queue,
		FilterName:           "test",
		DestinationARN:       "arn:aws:lambda:us-west-2:123456789012:function:example",
		LogGroupNamePrefixes: []string{"*"},
		NumWorkers:           4,
	})
	if err != nil {
		t.Fatal(err)
	}

	resp, err := h.HandleCleanupRequest(context.Background(), &subscriber.CleanupRequest{
		DeleteAll:              true,
		MaxGroupsPerInvocation: 100,
	})
	if err != nil {
		t.Fatal(err)
	}

	if got := resp.Subscription.Processed.Load(); got != total {
		t.Fatalf("processed=%d want=%d", got, total)
	}
	if got := len(queue.items); got != 0 {
		t.Fatalf("queued messages=%d want=0", got)
	}
}
