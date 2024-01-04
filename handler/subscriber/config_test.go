package subscriber_test

import (
	"fmt"
	"testing"

	"github.com/observeinc/aws-sam-apps/handler/handlertest"
	"github.com/observeinc/aws-sam-apps/handler/subscriber"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestConfig(t *testing.T) {
	testcases := []struct {
		subscriber.Config
		ExpectError error
	}{
		{
			ExpectError: subscriber.ErrMissingCloudWatchLogsClient,
		},
		{
			Config: subscriber.Config{
				CloudWatchLogsClient: &handlertest.CloudWatchLogsClient{},
				DestinationARN:       "hello",
			},
			ExpectError: subscriber.ErrMissingFilterName,
		},
		{
			Config: subscriber.Config{
				CloudWatchLogsClient: &handlertest.CloudWatchLogsClient{},
				FilterName:           "observe-logs-subscription",
				DestinationARN:       "hello",
			},
			ExpectError: subscriber.ErrInvalidARN,
		},
		{
			Config: subscriber.Config{
				CloudWatchLogsClient: &handlertest.CloudWatchLogsClient{},
				FilterName:           "observe-logs-subscription",
				DestinationARN:       "arn:aws:lambda:us-east-2:123456789012:function:my-function",
			},
		},
		{
			Config: subscriber.Config{
				CloudWatchLogsClient: &handlertest.CloudWatchLogsClient{},
				FilterName:           "observe-logs-subscription",
				RoleARN:              aws.String("arn:aws:iam::123456789012:role/test"),
			},
			ExpectError: subscriber.ErrMissingDestinationARN,
		},

		{
			Config: subscriber.Config{
				CloudWatchLogsClient: &handlertest.CloudWatchLogsClient{},
				FilterName:           "observe-logs-subscription",
				DestinationARN:       "arn:aws:firehose:us-west-2:123456789012:deliverystream/test",
				RoleARN:              aws.String("arn:aws:iam::123456789012:role/test"),
			},
		},
		{
			Config: subscriber.Config{
				CloudWatchLogsClient: &handlertest.CloudWatchLogsClient{},
				FilterName:           "observe-logs-subscription",
				LogGroupNamePatterns: []string{"!!"},
			},
			ExpectError: subscriber.ErrInvalidLogGroupName,
		},

		{
			Config: subscriber.Config{
				CloudWatchLogsClient: &handlertest.CloudWatchLogsClient{},
				FilterName:           "observe-logs-subscription",
				LogGroupNamePrefixes: []string{"\\"},
			},
			ExpectError: subscriber.ErrInvalidLogGroupName,
		},
		{
			Config: subscriber.Config{
				FilterName:           "ok",
				CloudWatchLogsClient: &handlertest.CloudWatchLogsClient{},
				LogGroupNamePrefixes: []string{"*"},
			},
		},
	}

	for i, tc := range testcases {
		tc := tc
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			err := tc.Validate()
			if diff := cmp.Diff(err, tc.ExpectError, cmpopts.EquateErrors()); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}

func TestLogFilter(t *testing.T) {
	testcases := []struct {
		subscriber.Config
		Matches map[string]bool
	}{
		{
			Config: subscriber.Config{
				LogGroupNamePatterns: []string{"prod"},
				DestinationARN:       "hello",
			},
			Matches: map[string]bool{
				"prod-1":  true,
				"eu-prod": true,
				"staging": false,
			},
		},
		{
			Config: subscriber.Config{
				LogGroupNamePatterns: []string{"prod"},
				LogGroupNamePrefixes: []string{"staging"},
				DestinationARN:       "hello",
			},
			Matches: map[string]bool{
				"prod-1":     true,
				"eu-prod":    true,
				"eu-staging": false,
				"staging-1":  true,
				"staging-2":  true,
				"dev-local":  false,
			},
		},
		{
			Config: subscriber.Config{
				LogGroupNamePatterns: []string{"prod", "*"},
				DestinationARN:       "hello",
			},
			Matches: map[string]bool{
				"prod-1":  true,
				"eu-prod": true,
				"staging": true,
			},
		},
	}

	for i, tc := range testcases {
		tc := tc
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			fn := tc.LogGroupFilter()
			for key, value := range tc.Matches {
				if fn(key) != value {
					t.Fatal(key)
				}
			}
		})
	}
}
