package subscriber_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
	"github.com/google/go-cmp/cmp"

	"github.com/observeinc/aws-sam-apps/pkg/handler/subscriber"
	"github.com/observeinc/aws-sam-apps/pkg/testing/awstest"
)

func TestHandleDiscovery(t *testing.T) {
	t.Parallel()

	client := &awstest.CloudWatchLogsClient{
		LogGroups: []types.LogGroup{
			{LogGroupName: aws.String("/aws/hello")},
			{LogGroupName: aws.String("/aws/ello")},
			{LogGroupName: aws.String("/aws/hola")},
		},
		SubscriptionFilters: []types.SubscriptionFilter{
			{LogGroupName: aws.String("/aws/hello")},
		},
	}

	testcases := []struct {
		DiscoveryRequest   *subscriber.DiscoveryRequest
		ExpectJSONResponse string
	}{
		{
			DiscoveryRequest: &subscriber.DiscoveryRequest{
				LogGroupNamePatterns: []*string{aws.String("*")},
			},
			/* matches:
			- /aws/hello
			- /aws/ello
			- /aws/hola
			*/
			ExpectJSONResponse: `{
				"discovery": {
					"logGroupCount": 3,
					"requestCount": 1,
					"subscription": {
						"deleted": 0,
						"updated": 0,
						"skipped": 0,
						"processed": 3
					}
				}
			}`,
		},
		{
			DiscoveryRequest: &subscriber.DiscoveryRequest{},
			/* matches nothing
			 */
			ExpectJSONResponse: `{
				"discovery": {
					"logGroupCount": 0,
					"requestCount": 0,
					"subscription": {
						"deleted": 0,
						"updated": 0,
						"skipped": 0,
						"processed": 0
					}
				}
			}`,
		},
		{
			DiscoveryRequest: &subscriber.DiscoveryRequest{
				LogGroupNamePrefixes: []*string{
					aws.String("/aws/he"),
					aws.String("/aws/ho"),
				},
			},
			/* matches:
			- /aws/hello
			- /aws/hola
			*/
			ExpectJSONResponse: `{
				"discovery": {
					"logGroupCount": 2,
					"requestCount": 2,
					"subscription": {
						"deleted": 0,
						"updated": 0,
						"skipped": 0,
						"processed": 2
					}
				}
			}`,
		},
		{
			DiscoveryRequest: &subscriber.DiscoveryRequest{
				LogGroupNamePatterns: []*string{
					aws.String("ello"),
					aws.String("foo"),
					aws.String("bar"),
				},
			},
			/* matches:
			- /aws/hello
			- /aws/ello
			*/
			ExpectJSONResponse: `{
				"discovery": {
					"logGroupCount": 2,
					"requestCount": 3,
					"subscription": {
						"deleted": 0,
						"updated": 0,
						"skipped": 0,
						"processed": 2
					}
				}
			}`,
		},
		{
			DiscoveryRequest: &subscriber.DiscoveryRequest{
				LogGroupNamePatterns: []*string{
					aws.String("ello"),
				},
				LogGroupNamePrefixes: []*string{
					aws.String("/aws/he"),
				},
			},
			/* matches:
			- /aws/hello
			- /aws/ello
			- /aws/hello
			*/
			ExpectJSONResponse: `{
				"discovery": {
					"logGroupCount": 3,
					"requestCount": 2,
					"subscription": {
						"deleted": 0,
						"updated": 0,
						"skipped": 0,
						"processed": 3
					}
				}
			}`,
		},
	}

	for i, tt := range testcases {
		tt := tt
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			t.Parallel()

			s, err := subscriber.New(&subscriber.Config{
				CloudWatchLogsClient: client,
				FilterName:           "test",
			})
			if err != nil {
				t.Fatal(err)
			}

			resp, err := s.HandleDiscoveryRequest(context.Background(), tt.DiscoveryRequest)
			if err != nil {
				t.Fatal(err)
			}

			var expect bytes.Buffer
			if err := json.Compact(&expect, []byte(tt.ExpectJSONResponse)); err != nil {
				t.Fatal(err)
			}
			got, err := json.Marshal(resp)
			if err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(expect.Bytes(), got); diff != "" {
				t.Error(diff)
			}
		})
	}
}

func TestHandleDiscoveryRequestUsesRequestFilterForPrune(t *testing.T) {
	t.Parallel()

	client := &awstest.CloudWatchLogsClient{
		LogGroups: []types.LogGroup{
			{LogGroupName: aws.String("/aws/lambda/app")},
			{LogGroupName: aws.String("/aws/ecs/svc")},
		},
		SubscriptionFilters: []types.SubscriptionFilter{
			{
				LogGroupName:           aws.String("/aws/lambda/app"),
				FilterName:             aws.String("test"),
				DestinationArn:         aws.String("arn:aws:firehose:us-east-1:123456789012:deliverystream/example"),
				FilterPattern:          aws.String(""),
				Distribution:           "",
				ApplyOnTransformedLogs: false,
			},
		},
	}

	h, err := subscriber.New(&subscriber.Config{
		CloudWatchLogsClient: client,
		FilterName:           "test",
		DestinationARN:       "arn:aws:firehose:us-east-1:123456789012:deliverystream/example",
		LogGroupNamePatterns: []string{"/aws/lambda/"},
		NumWorkers:           2,
	})
	if err != nil {
		t.Fatal(err)
	}

	inline := true
	resp, err := h.HandleDiscoveryRequest(context.Background(), &subscriber.DiscoveryRequest{
		LogGroupNamePatterns: []*string{aws.String("/aws/ecs")},
		FullyPrune:           true,
		Inline:               &inline,
	})
	if err != nil {
		t.Fatal(err)
	}

	if got := resp.Discovery.Subscription.Deleted.Load(); got != 1 {
		t.Fatalf("deleted=%d want=1", got)
	}
	if got := resp.Discovery.Subscription.Updated.Load(); got != 1 {
		t.Fatalf("updated=%d want=1", got)
	}
}

func TestHandleDiscoveryRequestEnqueuesContinuation(t *testing.T) {
	t.Parallel()

	const total = 120
	logGroups := make([]types.LogGroup, 0, total)
	for i := range total {
		name := fmt.Sprintf("/aws/lambda/discovery-%03d", i)
		logGroups = append(logGroups, types.LogGroup{
			LogGroupName: aws.String(name),
		})
	}

	queue := &queueRecorder{}
	client := &awstest.CloudWatchLogsClient{
		LogGroups: logGroups,
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

	inline := true
	resp, err := h.HandleDiscoveryRequest(context.Background(), &subscriber.DiscoveryRequest{
		LogGroupNamePrefixes:   []*string{aws.String("/aws/lambda/")},
		Inline:                 &inline,
		MaxGroupsPerInvocation: 50,
		JobID:                  "discover-job-1",
	})
	if err != nil {
		t.Fatal(err)
	}

	if got := resp.Discovery.LogGroupCount.Load(); got != 50 {
		t.Fatalf("logGroupCount=%d want=50", got)
	}
	if got := len(queue.items); got != 1 {
		t.Fatalf("queued messages=%d want=1", got)
	}

	req, ok := queue.items[0].(*subscriber.Request)
	if !ok || req == nil || req.DiscoveryRequest == nil {
		t.Fatalf("unexpected continuation payload type: %#v", queue.items[0])
	}
	if req.DiscoveryRequest.ScanToken == nil || *req.DiscoveryRequest.ScanToken == "" {
		t.Fatalf("continuation missing scanToken: %#v", req.DiscoveryRequest)
	}
	if req.ScanInputIndex != 0 {
		t.Fatalf("continuation scanInputIndex=%d want=0", req.ScanInputIndex)
	}
	if req.DiscoveryRequest.MaxGroupsPerInvocation != 50 {
		t.Fatalf("continuation maxGroups=%d want=50", req.DiscoveryRequest.MaxGroupsPerInvocation)
	}
}

func TestHandleDiscoveryRequestCompletesWithoutContinuation(t *testing.T) {
	t.Parallel()

	const total = 40
	logGroups := make([]types.LogGroup, 0, total)
	for i := range total {
		name := fmt.Sprintf("/aws/ecs/discovery-%03d", i)
		logGroups = append(logGroups, types.LogGroup{
			LogGroupName: aws.String(name),
		})
	}

	queue := &queueRecorder{}
	client := &awstest.CloudWatchLogsClient{
		LogGroups: logGroups,
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

	inline := true
	resp, err := h.HandleDiscoveryRequest(context.Background(), &subscriber.DiscoveryRequest{
		LogGroupNamePrefixes:   []*string{aws.String("/aws/ecs/")},
		Inline:                 &inline,
		MaxGroupsPerInvocation: 100,
	})
	if err != nil {
		t.Fatal(err)
	}

	if got := resp.Discovery.LogGroupCount.Load(); got != total {
		t.Fatalf("logGroupCount=%d want=%d", got, total)
	}
	if got := len(queue.items); got != 0 {
		t.Fatalf("queued messages=%d want=0", got)
	}
}
