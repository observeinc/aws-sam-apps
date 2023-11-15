package subscriber_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/observeinc/aws-sam-testing/handler/handlertest"
	"github.com/observeinc/aws-sam-testing/handler/subscriber"
)

type MockQueue struct {
	values []any
	sync.Mutex
}

func (m *MockQueue) Put(_ context.Context, vs ...any) error {
	m.Lock()
	defer m.Unlock()
	m.values = append(m.values, vs...)
	return nil
}

func TestHandleDiscovery(t *testing.T) {
	t.Parallel()

	client := &handlertest.CloudWatchLogsClient{
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
			DiscoveryRequest: &subscriber.DiscoveryRequest{},
			/* matches:
			- /aws/hello
			- /aws/ello
			- /aws/hola
			*/
			ExpectJSONResponse: `{
				"discovery": {
					"logGroupCount": 3,
					"requestCount": 1
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
					"requestCount": 2
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
					"requestCount": 3
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
					"requestCount": 2
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
				Queue:                &MockQueue{},
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

func TestHandleSubscribe(t *testing.T) {
	t.Parallel()

	client := &handlertest.CloudWatchLogsClient{
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
		SubscriptionRequest *subscriber.SubscriptionRequest
		ExpectJSONResponse  string
	}{
		{
			SubscriptionRequest: &subscriber.SubscriptionRequest{},
			ExpectJSONResponse: `{
				"subscription":	{
					"deleted": 0,
					"updated": 0,
					"skipped": 0,
					"processed": 0
				}
			}`,
		},
		{
			SubscriptionRequest: &subscriber.SubscriptionRequest{
				LogGroups: []*subscriber.LogGroup{
					{LogGroupName: "/aws/hello"},
				},
			},
			ExpectJSONResponse: `{
				"subscription":	{
					"deleted": 1,
					"updated": 0,
					"skipped": 0,
					"processed": 1
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
				Queue:                &MockQueue{},
			})
			if err != nil {
				t.Fatal(err)
			}

			resp, err := s.HandleSubscriptionRequest(context.Background(), tt.SubscriptionRequest)
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

func TestSubscriptionFilterDiff(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		Configure       types.SubscriptionFilter
		Existing        []types.SubscriptionFilter
		ExpectedActions []any
	}{
		{
			/*
				In the absence of a destination ARN, we delete all subscription
				filters that contain our filter name as a prefix.
			*/
			Configure: types.SubscriptionFilter{
				FilterName: aws.String("observe"),
			},
			Existing: []types.SubscriptionFilter{
				{
					FilterName: aws.String("foo"),
				},
				{
					FilterName: aws.String("observe-logs-subscription"),
				},
			},
			ExpectedActions: []any{
				&cloudwatchlogs.DeleteSubscriptionFilterInput{
					FilterName: aws.String("observe-logs-subscription"),
				},
			},
		},
		{
			Configure: types.SubscriptionFilter{
				FilterName:     aws.String("observe"),
				DestinationArn: aws.String("arn:aws:lambda:us-west-2:123456789012:function:example"),
			},
			Existing: []types.SubscriptionFilter{
				{
					FilterName: aws.String("foo"),
				},
				{
					FilterName: aws.String("observe-logs-subscription"),
				},
			},
			ExpectedActions: []any{
				&cloudwatchlogs.DeleteSubscriptionFilterInput{
					FilterName: aws.String("observe-logs-subscription"),
				},
				&cloudwatchlogs.PutSubscriptionFilterInput{
					FilterName:     aws.String("observe"),
					FilterPattern:  aws.String(""),
					DestinationArn: aws.String("arn:aws:lambda:us-west-2:123456789012:function:example"),
				},
			},
		},
		{
			/*
				Do nothing if we exceed the two subscription filter limit
			*/
			Configure: types.SubscriptionFilter{
				FilterName:     aws.String("observe"),
				DestinationArn: aws.String("arn:aws:lambda:us-west-2:123456789012:function:example"),
			},
			Existing: []types.SubscriptionFilter{},
			ExpectedActions: []any{
				&cloudwatchlogs.PutSubscriptionFilterInput{
					FilterName:     aws.String("observe"),
					FilterPattern:  aws.String(""),
					DestinationArn: aws.String("arn:aws:lambda:us-west-2:123456789012:function:example"),
				},
			},
		},
		{
			Configure: types.SubscriptionFilter{
				FilterName:     aws.String("observe"),
				DestinationArn: aws.String("arn:aws:lambda:us-west-2:123456789012:function:example"),
			},
			Existing: []types.SubscriptionFilter{
				{
					FilterName: aws.String("foo"),
				},
				{
					FilterName: aws.String("bar"),
				},
			},
			// no expected actions
		},
	}

	for i, tt := range testcases {
		tt := tt
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			t.Parallel()
			s, err := subscriber.New(
				&subscriber.Config{
					CloudWatchLogsClient: &handlertest.CloudWatchLogsClient{},
					Queue:                &MockQueue{},
					FilterName:           aws.ToString(tt.Configure.FilterName),
					DestinationARN:       aws.ToString(tt.Configure.DestinationArn),
					RoleARN:              aws.ToString(tt.Configure.RoleArn),
				})
			if err != nil {
				t.Fatal(err)
			}

			output := s.SubscriptionFilterDiff(tt.Existing)

			opts := cmpopts.IgnoreUnexported(
				cloudwatchlogs.PutSubscriptionFilterInput{},
				cloudwatchlogs.DeleteSubscriptionFilterInput{},
			)
			if diff := cmp.Diff(output, tt.ExpectedActions, opts); diff != "" {
				t.Error(diff)
			}
		})
	}
}
