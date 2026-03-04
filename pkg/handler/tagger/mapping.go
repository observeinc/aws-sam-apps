package tagger

import "strings"

// ResourceMapping describes how to map a CloudWatch metric namespace to an
// AWS resource type for tag lookups. ResourceType is the value expected by
// the Resource Groups Tagging API, and DimensionKey is the CloudWatch
// dimension that contains the resource identifier.
type ResourceMapping struct {
	ResourceType string
	DimensionKey string
}

// namespaceMappings maps CloudWatch metric namespaces to their corresponding
// Resource Groups Tagging API resource type and the dimension key that
// contains the resource identifier.
var namespaceMappings = map[string]ResourceMapping{
	"AWS/EC2":              {ResourceType: "ec2:instance", DimensionKey: "InstanceId"},
	"AWS/Lambda":           {ResourceType: "lambda:function", DimensionKey: "FunctionName"},
	"AWS/RDS":              {ResourceType: "rds:db", DimensionKey: "DBInstanceIdentifier"},
	"AWS/S3":               {ResourceType: "s3", DimensionKey: "BucketName"},
	"AWS/ELB":              {ResourceType: "elasticloadbalancing:loadbalancer", DimensionKey: "LoadBalancerName"},
	"AWS/ApplicationELB":   {ResourceType: "elasticloadbalancing:loadbalancer/app", DimensionKey: "LoadBalancer"},
	"AWS/NetworkELB":       {ResourceType: "elasticloadbalancing:loadbalancer/net", DimensionKey: "LoadBalancer"},
	"AWS/DynamoDB":         {ResourceType: "dynamodb:table", DimensionKey: "TableName"},
	"AWS/SQS":              {ResourceType: "sqs", DimensionKey: "QueueName"},
	"AWS/SNS":              {ResourceType: "sns", DimensionKey: "TopicName"},
	"AWS/ECS":              {ResourceType: "ecs:cluster", DimensionKey: "ClusterName"},
	"AWS/EBS":              {ResourceType: "ec2:volume", DimensionKey: "VolumeId"},
	"AWS/ElastiCache":      {ResourceType: "elasticache:cluster", DimensionKey: "CacheClusterId"},
	"AWS/Kinesis":          {ResourceType: "kinesis:stream", DimensionKey: "StreamName"},
	"AWS/Firehose":         {ResourceType: "firehose:deliverystream", DimensionKey: "DeliveryStreamName"},
	"AWS/ES":               {ResourceType: "es:domain", DimensionKey: "DomainName"},
	"AWS/ApiGateway":       {ResourceType: "apigateway", DimensionKey: "ApiName"},
	"AWS/StepFunctions":    {ResourceType: "states:stateMachine", DimensionKey: "StateMachineArn"},
	"AWS/NATGateway":       {ResourceType: "ec2:natgateway", DimensionKey: "NatGatewayId"},
	"AWS/Redshift":         {ResourceType: "redshift:cluster", DimensionKey: "ClusterIdentifier"},
	"AWS/CloudFront":       {ResourceType: "cloudfront:distribution", DimensionKey: "DistributionId"},
	"AWS/Route53":          {ResourceType: "route53:hostedzone", DimensionKey: "HostedZoneId"},
	"AWS/AutoScaling":      {ResourceType: "autoscaling:autoScalingGroup", DimensionKey: "AutoScalingGroupName"},
	"AWS/EFS":              {ResourceType: "elasticfilesystem:file-system", DimensionKey: "FileSystemId"},
	"AWS/Logs":             {ResourceType: "logs:log-group", DimensionKey: "LogGroupName"},
	"AWS/Events":           {ResourceType: "events:rule", DimensionKey: "RuleName"},
	"AWS/KinesisAnalytics": {ResourceType: "kinesisanalytics:application", DimensionKey: "Application"},
}

// LookupNamespace returns the resource mapping for a given CloudWatch
// namespace. Returns the mapping and true if found, zero value and false
// otherwise.
func LookupNamespace(namespace string) (ResourceMapping, bool) {
	m, ok := namespaceMappings[namespace]
	return m, ok
}

// AllResourceTypes returns the deduplicated set of resource type strings
// from the mapping table. Useful for bulk tag:GetResources calls.
func AllResourceTypes() []string {
	seen := make(map[string]struct{})
	var types []string
	for _, m := range namespaceMappings {
		if _, ok := seen[m.ResourceType]; !ok {
			seen[m.ResourceType] = struct{}{}
			types = append(types, m.ResourceType)
		}
	}
	return types
}

// ExtractResourceID extracts the resource identifier from an ARN. For most
// resources this is the segment after the last '/' or ':'.
func ExtractResourceID(arn string) string {
	if i := strings.LastIndex(arn, "/"); i >= 0 {
		return arn[i+1:]
	}
	if i := strings.LastIndex(arn, ":"); i >= 0 {
		return arn[i+1:]
	}
	return arn
}
