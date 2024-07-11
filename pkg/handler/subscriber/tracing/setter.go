package tracing

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/smithy-go/middleware"
	"go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws"
	"go.opentelemetry.io/otel/attribute"
)

var AttributeSetters = []otelaws.AttributeSetter{AttributeSetter}

func AttributeSetter(_ context.Context, in middleware.InitializeInput) (attrs []attribute.KeyValue) {
	switch v := in.Parameters.(type) {
	case *cloudwatchlogs.DescribeLogGroupsInput:
		if s := v.LogGroupNamePattern; s != nil {
			attrs = append(attrs, attribute.String("log_group_name_pattern", aws.ToString(s)))
		}
		if s := v.LogGroupNamePrefix; s != nil {
			attrs = append(attrs, attribute.String("log_group_name_prefix", aws.ToString(s)))
		}
	case *cloudwatchlogs.DescribeSubscriptionFiltersInput:
		attrs = append(attrs, attribute.String("log_group_name", aws.ToString(v.LogGroupName)))
		if s := v.FilterNamePrefix; s != nil {
			attrs = append(attrs, attribute.String("filter_name_prefix", aws.ToString(s)))
		}
	case *cloudwatchlogs.PutSubscriptionFilterInput:
		attrs = append(attrs,
			attribute.String("destination_arn", aws.ToString(v.DestinationArn)),
			attribute.String("log_group_name", aws.ToString(v.LogGroupName)),
			attribute.String("role_arn", aws.ToString(v.RoleArn)),
			attribute.String("filter_name", aws.ToString(v.FilterName)),
		)
	case *cloudwatchlogs.DeleteSubscriptionFilterInput:
		attrs = append(attrs,
			attribute.String("log_group_name", aws.ToString(v.LogGroupName)),
			attribute.String("filter_name", aws.ToString(v.FilterName)),
		)
	}
	return
}
