package subscriber_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/observeinc/aws-sam-testing/handler/handlertest"
	"github.com/observeinc/aws-sam-testing/handler/subscriber"
)

func TestHandleSubscribe(t *testing.T) {
	t.Parallel()

	client := &handlertest.CloudWatchLogsClient{
		LogGroups: []types.LogGroup{
			{LogGroupName: aws.String("/aws/hello")},
			{LogGroupName: aws.String("/aws/ello")},
			{LogGroupName: aws.String("/aws/hola")},
		},
		SubscriptionFilters: []types.SubscriptionFilter{
			{
				FilterName:   aws.String("test"),
				LogGroupName: aws.String("/aws/hello"),
			},
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
					{LogGroupName: "/aws/ello"},
				},
			},
			ExpectJSONResponse: `{
				"subscription":	{
					"deleted": 1,
					"updated": 0,
					"skipped": 1,
					"processed": 2
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
				LogGroupNamePrefixes: []string{"/aws/h"},
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
