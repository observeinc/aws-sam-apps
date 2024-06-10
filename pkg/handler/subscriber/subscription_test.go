package subscriber_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/observeinc/aws-sam-apps/pkg/handler/subscriber"
	"github.com/observeinc/aws-sam-apps/pkg/testing/awstest"
)

func TestHandleSubscribe(t *testing.T) {
	t.Parallel()

	client := &awstest.CloudWatchLogsClient{
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
			Configure: types.SubscriptionFilter{
				FilterName:     aws.String("observe"),
				DestinationArn: aws.String("arn:aws:firehose:us-west-2:123456789012:deliverystream/example"),
				RoleArn:        aws.String("arn:aws:iam::123456789012:role/example"),
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
					DestinationArn: aws.String("arn:aws:firehose:us-west-2:123456789012:deliverystream/example"),
					RoleArn:        aws.String("arn:aws:iam::123456789012:role/example"),
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
					CloudWatchLogsClient: &awstest.CloudWatchLogsClient{},
					FilterName:           aws.ToString(tt.Configure.FilterName),
					DestinationARN:       aws.ToString(tt.Configure.DestinationArn),
					RoleARN:              tt.Configure.RoleArn,
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

var errTooManyConcurrentRequests = errors.New("too many concurrent requests")

// TestHandleSubscribeConcurrent verifies `NumWorkers` parameter works as intended.
func TestHandleSubscribeConcurrent(t *testing.T) {
	t.Parallel()

	// Create a client that errors if invoked concurrently beyond a given amount
	cappedClient := func(t *testing.T, capacity int, delay time.Duration) subscriber.CloudWatchLogsClient {
		t.Helper()
		semaphore := make(chan struct{}, capacity)
		t.Cleanup(func() { close(semaphore) })
		return &awstest.CloudWatchLogsClient{
			DescribeSubscriptionFiltersFunc: func(context.Context, *cloudwatchlogs.DescribeSubscriptionFiltersInput, ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.DescribeSubscriptionFiltersOutput, error) {
				select {
				case semaphore <- struct{}{}:
					t.Log("acquired semaphore")
					defer func() {
						t.Log("releasing semaphore")
						<-semaphore
					}()
				default:
					// we were unable to acquire semaphore, there must be too many
					// concurrent invocations of this method
					return nil, errTooManyConcurrentRequests
				}
				// hold semaphore up
				// yes, using time is poor form, but introducing a channel to
				// tightly control execution flow seems overkill for now.
				<-time.After(delay)
				return &cloudwatchlogs.DescribeSubscriptionFiltersOutput{}, nil
			},
		}
	}

	subscriptionRequest := &subscriber.SubscriptionRequest{
		LogGroups: []*subscriber.LogGroup{
			{LogGroupName: "/aws/hello"},
			{LogGroupName: "/aws/ello"},
			{LogGroupName: "/aws/llo"},
			{LogGroupName: "/aws/lo"},
			{LogGroupName: "/aws/o"},
		},
	}

	testcases := []struct {
		NumWorkers     int
		ClientCapacity int
		ClientDelay    time.Duration
		ExpectError    error
	}{
		{
			NumWorkers:     3,
			ClientCapacity: 3,
			ClientDelay:    time.Second,
		},
		{
			NumWorkers:     3,
			ClientCapacity: 2,
			ClientDelay:    time.Second,
			ExpectError:    errTooManyConcurrentRequests,
		},
	}

	for i, tt := range testcases {
		tt := tt
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			t.Parallel()

			s, err := subscriber.New(&subscriber.Config{
				CloudWatchLogsClient: cappedClient(t, tt.ClientCapacity, tt.ClientDelay),
				FilterName:           "test",
				NumWorkers:           tt.NumWorkers,
			})
			if err != nil {
				t.Fatal(err)
			}

			_, err = s.HandleSubscriptionRequest(context.Background(), subscriptionRequest)
			if diff := cmp.Diff(err, tt.ExpectError, cmpopts.EquateErrors()); diff != "" {
				t.Error(diff)
			}
		})
	}
}
