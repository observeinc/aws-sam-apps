package handlertest

import (
	"context"
	"errors"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
)

var errNoLogGroup = errors.New("no log group provided")

type CloudWatchLogsClient struct {
	// list of log groups and subscription filters to use
	LogGroups           []types.LogGroup
	SubscriptionFilters []types.SubscriptionFilter

	// optionally override functions
	DescribeLogGroupsFunc           func(context.Context, *cloudwatchlogs.DescribeLogGroupsInput, ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.DescribeLogGroupsOutput, error)
	DescribeSubscriptionFiltersFunc func(context.Context, *cloudwatchlogs.DescribeSubscriptionFiltersInput, ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.DescribeSubscriptionFiltersOutput, error)
	PutSubscriptionFilterFunc       func(context.Context, *cloudwatchlogs.PutSubscriptionFilterInput, ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.PutSubscriptionFilterOutput, error)
	DeleteSubscriptionFilterFunc    func(context.Context, *cloudwatchlogs.DeleteSubscriptionFilterInput, ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.DeleteSubscriptionFilterOutput, error)
}

func (c *CloudWatchLogsClient) DescribeLogGroups(ctx context.Context, input *cloudwatchlogs.DescribeLogGroupsInput, opts ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.DescribeLogGroupsOutput, error) {
	if c.DescribeLogGroupsFunc != nil {
		return c.DescribeLogGroupsFunc(ctx, input, opts...)
	}
	var output cloudwatchlogs.DescribeLogGroupsOutput
	nextToken := input.NextToken

logGroups:
	for _, logGroup := range c.LogGroups {
		if nextToken != nil && aws.ToString(nextToken) != aws.ToString(logGroup.LogGroupName) {
			continue
		}
		nextToken = nil

		switch {
		case input.Limit != nil && len(output.LogGroups) >= int(*input.Limit):
			output.NextToken = logGroup.LogGroupName
			break logGroups
		case !strings.HasPrefix(aws.ToString(logGroup.LogGroupName), aws.ToString(input.LogGroupNamePrefix)):
			continue
		case !strings.Contains(aws.ToString(logGroup.LogGroupName), aws.ToString(input.LogGroupNamePattern)):
			continue
		default:
			output.LogGroups = append(output.LogGroups, logGroup)
		}
	}
	return &output, nil
}

func (c *CloudWatchLogsClient) DescribeSubscriptionFilters(ctx context.Context, input *cloudwatchlogs.DescribeSubscriptionFiltersInput, opts ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.DescribeSubscriptionFiltersOutput, error) {
	if c.DescribeSubscriptionFiltersFunc != nil {
		return c.DescribeSubscriptionFiltersFunc(ctx, input, opts...)
	}

	if input == nil || input.LogGroupName == nil {
		return nil, errNoLogGroup
	}

	output := &cloudwatchlogs.DescribeSubscriptionFiltersOutput{
		SubscriptionFilters: []types.SubscriptionFilter{},
	}

	for _, logGroup := range c.LogGroups {
		if aws.ToString(input.LogGroupName) == aws.ToString(logGroup.LogGroupName) {
			for _, subscriptionFilter := range c.SubscriptionFilters {
				if aws.ToString(input.LogGroupName) != aws.ToString(subscriptionFilter.LogGroupName) {
					continue
				}
				if !strings.HasPrefix(aws.ToString(subscriptionFilter.FilterName), aws.ToString(input.FilterNamePrefix)) {
					continue
				}
				output.SubscriptionFilters = append(output.SubscriptionFilters, subscriptionFilter)
			}
			return output, nil
		}
	}

	// TODO: need to verify what the correct error to surface here is.
	return nil, &types.ResourceNotFoundException{Message: aws.String("log group not found")}
}

func (c *CloudWatchLogsClient) PutSubscriptionFilter(ctx context.Context, input *cloudwatchlogs.PutSubscriptionFilterInput, opts ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.PutSubscriptionFilterOutput, error) {
	if c.PutSubscriptionFilterFunc != nil {
		return c.PutSubscriptionFilterFunc(ctx, input, opts...)
	}
	return nil, nil
}

func (c *CloudWatchLogsClient) DeleteSubscriptionFilter(ctx context.Context, input *cloudwatchlogs.DeleteSubscriptionFilterInput, opts ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.DeleteSubscriptionFilterOutput, error) {
	if c.DeleteSubscriptionFilterFunc != nil {
		return c.DeleteSubscriptionFilterFunc(ctx, input, opts...)
	}
	return nil, nil
}
