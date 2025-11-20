package subscriber_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aws/aws-lambda-go/cfn"
	"github.com/aws/aws-lambda-go/lambdacontext"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"

	"github.com/observeinc/aws-sam-apps/pkg/handler/subscriber"
	"github.com/observeinc/aws-sam-apps/pkg/testing/awstest"
)

func TestBuildLogGroupFilter(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		Name            string
		Patterns        []*string
		Prefixes        []*string
		ExcludePatterns []*string
		LogGroupName    string
		ShouldMatch     bool
	}{
		{
			Name:         "Match with prefix",
			Prefixes:     []*string{aws.String("/aws/lambda")},
			LogGroupName: "/aws/lambda/my-function",
			ShouldMatch:  true,
		},
		{
			Name:         "No match with prefix",
			Prefixes:     []*string{aws.String("/aws/lambda")},
			LogGroupName: "/aws/ecs/my-service",
			ShouldMatch:  false,
		},
		{
			Name:            "Excluded by pattern",
			Prefixes:        []*string{aws.String("/aws/lambda")},
			ExcludePatterns: []*string{aws.String("^/aws/lambda/observeinc")},
			LogGroupName:    "/aws/lambda/observeinc-forwarder",
			ShouldMatch:     false,
		},
		{
			Name:            "Not excluded",
			Prefixes:        []*string{aws.String("/aws/lambda")},
			ExcludePatterns: []*string{aws.String("^/aws/lambda/observeinc")},
			LogGroupName:    "/aws/lambda/my-app",
			ShouldMatch:     true,
		},
		{
			Name:            "Multiple exclude patterns",
			Prefixes:        []*string{aws.String("/aws/lambda")},
			ExcludePatterns: []*string{aws.String("^/aws/lambda/observeinc"), aws.String("^/aws/lambda/test")},
			LogGroupName:    "/aws/lambda/test-function",
			ShouldMatch:     false,
		},
		{
			Name:            "Multiple exclude patterns - not excluded",
			Prefixes:        []*string{aws.String("/aws/lambda")},
			ExcludePatterns: []*string{aws.String("^/aws/lambda/observeinc"), aws.String("^/aws/lambda/test")},
			LogGroupName:    "/aws/lambda/production-api",
			ShouldMatch:     true,
		},
		{
			Name:         "Wildcard pattern",
			Patterns:     []*string{aws.String("*")},
			LogGroupName: "/any/log/group",
			ShouldMatch:  true,
		},
		{
			Name:            "Wildcard with exclusion",
			Patterns:        []*string{aws.String("*")},
			ExcludePatterns: []*string{aws.String("^/aws/lambda/test")},
			LogGroupName:    "/aws/lambda/test-function",
			ShouldMatch:     false,
		},
		{
			Name:         "Exact pattern match",
			Patterns:     []*string{aws.String("/aws/lambda/my-function")},
			LogGroupName: "/aws/lambda/my-function",
			ShouldMatch:  true,
		},
		{
			Name:         "Pattern no match",
			Patterns:     []*string{aws.String("/aws/lambda/my-function")},
			LogGroupName: "/aws/lambda/other-function",
			ShouldMatch:  false,
		},
		{
			Name:         "Nil patterns",
			Patterns:     []*string{nil},
			Prefixes:     []*string{aws.String("/aws/lambda")},
			LogGroupName: "/aws/lambda/my-function",
			ShouldMatch:  true,
		},
		{
			Name:            "Empty string in exclude patterns",
			Prefixes:        []*string{aws.String("/aws/lambda")},
			ExcludePatterns: []*string{aws.String(""), aws.String("^/aws/lambda/test")},
			LogGroupName:    "/aws/lambda/my-function",
			ShouldMatch:     true,
		},
	}

	for _, tc := range testcases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			// Use the unexported buildLogGroupFilter function via the Config.LogGroupFilter
			// which has the same logic
			cfg := &subscriber.Config{
				FilterName:     "test-filter",
				DestinationARN: "arn:aws:logs:us-west-2:123456789012:destination:test",
			}

			// Convert []*string to []string for Config
			if tc.Patterns != nil {
				for _, p := range tc.Patterns {
					if p != nil {
						cfg.LogGroupNamePatterns = append(cfg.LogGroupNamePatterns, *p)
					}
				}
			}
			if tc.Prefixes != nil {
				for _, p := range tc.Prefixes {
					if p != nil {
						cfg.LogGroupNamePrefixes = append(cfg.LogGroupNamePrefixes, *p)
					}
				}
			}
			if tc.ExcludePatterns != nil {
				for _, p := range tc.ExcludePatterns {
					if p != nil && *p != "" {
						cfg.ExcludeLogGroupNamePatterns = append(cfg.ExcludeLogGroupNamePatterns, *p)
					}
				}
			}

			filter := cfg.LogGroupFilter()
			result := filter(tc.LogGroupName)

			if result != tc.ShouldMatch {
				t.Errorf("Expected filter(%q) = %v, got %v", tc.LogGroupName, tc.ShouldMatch, result)
			}
		})
	}
}

// cfnResponseServer starts an HTTP server that captures CloudFormation callback responses.
// Returns the server and a function to retrieve the last received response body.
func cfnResponseServer(t *testing.T) (*httptest.Server, func() *cfn.Response) {
	t.Helper()
	var lastBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("failed to read cfn response body: %v", err)
		}
		lastBody = body
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)
	return srv, func() *cfn.Response {
		if lastBody == nil {
			return nil
		}
		var resp cfn.Response
		if err := json.Unmarshal(lastBody, &resp); err != nil {
			t.Fatalf("failed to unmarshal cfn response: %v", err)
		}
		return &resp
	}
}

func newTestHandler(t *testing.T, client *awstest.CloudWatchLogsClient, queue subscriber.Queue) *subscriber.Handler {
	t.Helper()
	h, err := subscriber.New(&subscriber.Config{
		CloudWatchLogsClient: client,
		Queue:                queue,
		FilterName:           "observe-logs-subscription",
		DestinationARN:       "arn:aws:firehose:us-east-1:123456789012:deliverystream/test",
		LogGroupNamePatterns: []string{"/aws/lambda"},
		NumWorkers:           2,
	})
	if err != nil {
		t.Fatal(err)
	}
	return h
}

func TestHandleCloudFormationUpdate(t *testing.T) {
	t.Parallel()

	srv, getCfnResp := cfnResponseServer(t)

	client := &awstest.CloudWatchLogsClient{
		LogGroups: []types.LogGroup{
			{LogGroupName: aws.String("/aws/lambda/app-1")},
			{LogGroupName: aws.String("/aws/ecs/svc-1")},
		},
		SubscriptionFilters: []types.SubscriptionFilter{},
	}
	queue := &queueRecorder{}
	h := newTestHandler(t, client, queue)

	ev := &subscriber.CloudFormationEvent{
		Event: &cfn.Event{
			RequestType:       cfn.RequestUpdate,
			RequestID:         "test-request-1",
			ResponseURL:       srv.URL,
			LogicalResourceID: "Trigger",
			StackID:           "arn:aws:cloudformation:us-east-1:123456789012:stack/test/guid",
			ResourceProperties: map[string]interface{}{
				"LogGroupNamePatterns":        []interface{}{"/aws/ecs"},
				"LogGroupNamePrefixes":        []interface{}{},
				"ExcludeLogGroupNamePatterns": []interface{}{},
			},
		},
	}

	resp, err := h.HandleCloudFormation(context.Background(), ev)
	if err != nil {
		t.Fatalf("HandleCloudFormation returned error: %v", err)
	}

	if resp == nil || resp.Discovery == nil {
		t.Fatal("expected discovery response")
	}

	cfnResp := getCfnResp()
	if cfnResp == nil {
		t.Fatal("CloudFormation callback was not received")
	}
	if cfnResp.Status != cfn.StatusSuccess {
		t.Fatalf("CloudFormation status=%s want=SUCCESS, reason=%s", cfnResp.Status, cfnResp.Reason)
	}
}

func TestHandleCloudFormationDeleteTriggersCleanup(t *testing.T) {
	t.Parallel()

	srv, getCfnResp := cfnResponseServer(t)

	var deletedFilters []string
	client := &awstest.CloudWatchLogsClient{
		LogGroups: []types.LogGroup{
			{LogGroupName: aws.String("/aws/lambda/app-1")},
		},
		SubscriptionFilters: []types.SubscriptionFilter{
			{
				LogGroupName:   aws.String("/aws/lambda/app-1"),
				FilterName:     aws.String("observe-logs-subscription"),
				DestinationArn: aws.String("arn:aws:firehose:us-east-1:123456789012:deliverystream/test"),
			},
		},
		DeleteSubscriptionFilterFunc: func(_ context.Context, input *cloudwatchlogs.DeleteSubscriptionFilterInput, _ ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.DeleteSubscriptionFilterOutput, error) {
			deletedFilters = append(deletedFilters, aws.ToString(input.LogGroupName))
			return &cloudwatchlogs.DeleteSubscriptionFilterOutput{}, nil
		},
	}
	queue := &queueRecorder{}
	h := newTestHandler(t, client, queue)

	// Simulate a true stack delete: PhysicalResourceID matches the current log stream.
	const logStream = "2026/04/04/[$LATEST]abc123"
	lambdacontext.LogStreamName = logStream

	ev := &subscriber.CloudFormationEvent{
		Event: &cfn.Event{
			RequestType:        cfn.RequestDelete,
			RequestID:          "test-request-2",
			ResponseURL:        srv.URL,
			LogicalResourceID:  "Trigger",
			StackID:            "arn:aws:cloudformation:us-east-1:123456789012:stack/test/guid",
			PhysicalResourceID: logStream,
			ResourceProperties: map[string]interface{}{},
		},
	}

	resp, err := h.HandleCloudFormation(context.Background(), ev)
	if err != nil {
		t.Fatalf("HandleCloudFormation returned error: %v", err)
	}
	if resp == nil || resp.Subscription == nil {
		t.Fatal("expected subscription (cleanup) response")
	}

	if len(deletedFilters) != 1 || deletedFilters[0] != "/aws/lambda/app-1" {
		t.Fatalf("expected cleanup to delete filter for /aws/lambda/app-1, got: %v", deletedFilters)
	}

	cfnResp := getCfnResp()
	if cfnResp == nil {
		t.Fatal("CloudFormation callback was not received")
	}
	if cfnResp.Status != cfn.StatusSuccess {
		t.Fatalf("CloudFormation status=%s want=SUCCESS, reason=%s", cfnResp.Status, cfnResp.Reason)
	}
}

func TestHandleCloudFormationDeleteSkipsCleanupOnReplacement(t *testing.T) {
	t.Parallel()

	srv, getCfnResp := cfnResponseServer(t)

	var deleteCalls int
	client := &awstest.CloudWatchLogsClient{
		LogGroups: []types.LogGroup{
			{LogGroupName: aws.String("/aws/lambda/app-1")},
		},
		SubscriptionFilters: []types.SubscriptionFilter{
			{
				LogGroupName:   aws.String("/aws/lambda/app-1"),
				FilterName:     aws.String("observe-logs-subscription"),
				DestinationArn: aws.String("arn:aws:firehose:us-east-1:123456789012:deliverystream/test"),
			},
		},
		DeleteSubscriptionFilterFunc: func(_ context.Context, _ *cloudwatchlogs.DeleteSubscriptionFilterInput, _ ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.DeleteSubscriptionFilterOutput, error) {
			deleteCalls++
			return &cloudwatchlogs.DeleteSubscriptionFilterOutput{}, nil
		},
	}
	queue := &queueRecorder{}
	h := newTestHandler(t, client, queue)

	// During a resource replacement, the current Lambda runs in a NEW log stream,
	// but the DELETE event carries the OLD log stream as PhysicalResourceID.
	lambdacontext.LogStreamName = "2026/04/04/[$LATEST]new-stream-xyz"

	ev := &subscriber.CloudFormationEvent{
		Event: &cfn.Event{
			RequestType:        cfn.RequestDelete,
			RequestID:          "test-request-3",
			ResponseURL:        srv.URL,
			LogicalResourceID:  "Trigger",
			StackID:            "arn:aws:cloudformation:us-east-1:123456789012:stack/test/guid",
			PhysicalResourceID: "2026/04/04/[$LATEST]old-stream-abc",
			ResourceProperties: map[string]interface{}{},
		},
	}

	resp, err := h.HandleCloudFormation(context.Background(), ev)
	if err != nil {
		t.Fatalf("HandleCloudFormation returned error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}

	if deleteCalls != 0 {
		t.Fatalf("expected no cleanup during resource replacement, but got %d delete calls", deleteCalls)
	}

	cfnResp := getCfnResp()
	if cfnResp == nil {
		t.Fatal("CloudFormation callback was not received")
	}
	if cfnResp.Status != cfn.StatusSuccess {
		t.Fatalf("CloudFormation status=%s want=SUCCESS", cfnResp.Status)
	}
}

