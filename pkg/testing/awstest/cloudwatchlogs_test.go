package awstest_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/observeinc/aws-sam-apps/pkg/testing/awstest"
)

func TestCloudWatchLogsDescribeLogGroups(t *testing.T) {
	client := &awstest.CloudWatchLogsClient{
		LogGroups: []types.LogGroup{
			{LogGroupName: aws.String("/aws/hello")},
			{LogGroupName: aws.String("/aws/ello")},
			{LogGroupName: aws.String("/aws/hola")},
		},
	}

	testcases := []struct {
		Input        *cloudwatchlogs.DescribeLogGroupsInput
		ExpectOutput *cloudwatchlogs.DescribeLogGroupsOutput
	}{
		{
			Input: &cloudwatchlogs.DescribeLogGroupsInput{},
			ExpectOutput: &cloudwatchlogs.DescribeLogGroupsOutput{
				LogGroups: []types.LogGroup{
					{LogGroupName: aws.String("/aws/hello")},
					{LogGroupName: aws.String("/aws/ello")},
					{LogGroupName: aws.String("/aws/hola")},
				},
			},
		},
		{
			Input: &cloudwatchlogs.DescribeLogGroupsInput{
				LogGroupNamePrefix: aws.String("/aws/h"),
			},
			ExpectOutput: &cloudwatchlogs.DescribeLogGroupsOutput{
				LogGroups: []types.LogGroup{
					{LogGroupName: aws.String("/aws/hello")},
					{LogGroupName: aws.String("/aws/hola")},
				},
			},
		},
		{
			Input: &cloudwatchlogs.DescribeLogGroupsInput{
				LogGroupNamePattern: aws.String("ell"),
			},
			ExpectOutput: &cloudwatchlogs.DescribeLogGroupsOutput{
				LogGroups: []types.LogGroup{
					{LogGroupName: aws.String("/aws/hello")},
					{LogGroupName: aws.String("/aws/ello")},
				},
			},
		},
		{
			Input: &cloudwatchlogs.DescribeLogGroupsInput{
				Limit: aws.Int32(1),
			},
			ExpectOutput: &cloudwatchlogs.DescribeLogGroupsOutput{
				LogGroups: []types.LogGroup{
					{LogGroupName: aws.String("/aws/hello")},
				},
				NextToken: aws.String("/aws/ello"),
			},
		},
		{
			Input: &cloudwatchlogs.DescribeLogGroupsInput{
				Limit:     aws.Int32(1),
				NextToken: aws.String("/aws/ello"),
			},
			ExpectOutput: &cloudwatchlogs.DescribeLogGroupsOutput{
				LogGroups: []types.LogGroup{
					{LogGroupName: aws.String("/aws/ello")},
				},
				NextToken: aws.String("/aws/hola"),
			},
		},
	}

	for i, tt := range testcases {
		tt := tt
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			output, err := client.DescribeLogGroups(context.Background(), tt.Input)
			if err != nil {
				t.Fatal(err)
			}

			jsonExpect, err := json.MarshalIndent(tt.ExpectOutput, "  ", "")
			if err != nil {
				t.Fatal("failed to marshal expected output", err)
			}

			jsonOutput, err := json.MarshalIndent(output, "  ", "")
			if err != nil {
				t.Fatal("failed to marshal returned output", err)
			}

			if diff := cmp.Diff(jsonExpect, jsonOutput); diff != "" {
				t.Error(diff)
			}
		})
	}
}

func TestCloudWatchLogsDescribeSubscriptionFilters(t *testing.T) {
	client := &awstest.CloudWatchLogsClient{
		LogGroups: []types.LogGroup{
			{LogGroupName: aws.String("/aws/hello")},
		},
		SubscriptionFilters: []types.SubscriptionFilter{
			{
				FilterName:   aws.String("example-1"),
				LogGroupName: aws.String("/aws/hello"),
			},
			{
				FilterName:   aws.String("example-2"),
				LogGroupName: aws.String("/aws/hello"),
			},
		},
	}

	testcases := []struct {
		Input        *cloudwatchlogs.DescribeSubscriptionFiltersInput
		ExpectError  error
		ExpectOutput *cloudwatchlogs.DescribeSubscriptionFiltersOutput
	}{
		{
			Input: &cloudwatchlogs.DescribeSubscriptionFiltersInput{
				LogGroupName: aws.String("no-match"),
			},
			ExpectError: cmpopts.AnyError,
		},
		{
			Input: &cloudwatchlogs.DescribeSubscriptionFiltersInput{
				LogGroupName: aws.String("/aws/hello"),
			},
			ExpectOutput: &cloudwatchlogs.DescribeSubscriptionFiltersOutput{
				SubscriptionFilters: []types.SubscriptionFilter{
					{
						FilterName:   aws.String("example-1"),
						LogGroupName: aws.String("/aws/hello"),
					},
					{
						FilterName:   aws.String("example-2"),
						LogGroupName: aws.String("/aws/hello"),
					},
				},
			},
		},
		{
			Input: &cloudwatchlogs.DescribeSubscriptionFiltersInput{
				LogGroupName:     aws.String("/aws/hello"),
				FilterNamePrefix: aws.String("exam"),
			},
			ExpectOutput: &cloudwatchlogs.DescribeSubscriptionFiltersOutput{
				SubscriptionFilters: []types.SubscriptionFilter{
					{
						FilterName:   aws.String("example-1"),
						LogGroupName: aws.String("/aws/hello"),
					},
					{
						FilterName:   aws.String("example-2"),
						LogGroupName: aws.String("/aws/hello"),
					},
				},
			},
		},
		{
			Input: &cloudwatchlogs.DescribeSubscriptionFiltersInput{
				LogGroupName:     aws.String("/aws/hello"),
				FilterNamePrefix: aws.String("exam-"),
			},
			ExpectOutput: &cloudwatchlogs.DescribeSubscriptionFiltersOutput{
				SubscriptionFilters: []types.SubscriptionFilter{},
			},
		},
	}

	for i, tt := range testcases {
		tt := tt
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			output, err := client.DescribeSubscriptionFilters(context.Background(), tt.Input)
			if diff := cmp.Diff(err, tt.ExpectError, cmpopts.EquateErrors()); diff != "" {
				t.Fatal(diff)
			}

			jsonExpect, err := json.MarshalIndent(tt.ExpectOutput, "  ", "")
			if err != nil {
				t.Fatal("failed to marshal expected output", err)
			}

			jsonOutput, err := json.MarshalIndent(output, "  ", "")
			if err != nil {
				t.Fatal("failed to marshal returned output", err)
			}

			if diff := cmp.Diff(jsonExpect, jsonOutput); diff != "" {
				t.Error(diff)
			}
		})
	}
}
