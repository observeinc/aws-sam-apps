package handlertest

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
)

type CloudWatchLogsClient struct {
	DescribeLogGroupsFunc           func(context.Context, *cloudwatchlogs.DescribeLogGroupsInput, ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.DescribeLogGroupsOutput, error)
	DescribeSubscriptionFiltersFunc func(context.Context, *cloudwatchlogs.DescribeSubscriptionFiltersInput, ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.DescribeSubscriptionFiltersOutput, error)
}

func (c *CloudWatchLogsClient) DescribeLogGroups(ctx context.Context, input *cloudwatchlogs.DescribeLogGroupsInput, optFns ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.DescribeLogGroupsOutput, error) {
	if c.DescribeLogGroupsFunc == nil {
		return nil, nil
	}

	return c.DescribeLogGroupsFunc(ctx, input, optFns...)
}

func (c *CloudWatchLogsClient) DescribeSubscriptionFilters(ctx context.Context, input *cloudwatchlogs.DescribeSubscriptionFiltersInput, optFns ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.DescribeSubscriptionFiltersOutput, error) {
	if c.DescribeSubscriptionFiltersFunc == nil {
		return nil, nil
	}
	return c.DescribeSubscriptionFiltersFunc(ctx, input, optFns...)
}
