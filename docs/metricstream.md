# Observe CloudWatch Metrics Stream

The Observe CloudWatch Metrics Stream application delivers CloudWatch Metrics to an Amazon S3 Bucket.

## Configuration Parameters

The application is configurable through several parameters that determine how data is buffered and delivered:

- **BucketARN**: The ARN of the S3 bucket where CloudWatch metrics will be delivered
- **Prefix**: An optional prefix path within the S3 bucket where records will be stored.
- **OutputFormat**: the output format for metrics. One of `json`, `opentelemetry0.7` or `opentelemetry1.0`.
- **NameOverride**: If specified, sets the name of the Firehose Delivery Stream; otherwise, the stack name is used.
- **BufferingInterval**: The amount of time Firehose buffers incoming data before delivering it (minimum 60 seconds, maximum 900 seconds).
- **BufferingSize**: The size of the buffer, in MiBs, that Firehose accumulates before delivering data (minimum 1 MiB, maximum 64 MiBs).
- **MetricStreamFilterURI**: An S3 URI containing a filter for collected metrics.

## Resources Created

The CloudWatch Metrics Stream application provisions the following AWS resources:

- **IAM Role**: Grants the Firehose service permission to access source and destination services.
- **CloudWatch Log Group**: Captures logging information from the Firehose delivery stream.
- **CloudWatch Log Stream**: A specific log stream for storing Firehose delivery logs.
- **Kinesis Firehose Delivery Stream**: The core component that manages the delivery of data to the S3 bucket.
- **CloudWatch Metrics Stream**: the component responsible for writing metrics to Kinesis Firehose.


## Filtering metrics

This module requires a URI to a pubicly readable S3 object containing a YAML or JSON definition
for what metrics to collect. Observe hosts some boilerplate filters you can use:

- `s3://observeinc/cloudwatchmetrics/filters/full.yaml` collects all metrics.
- `s3://observeinc/cloudwatchmetrics/filters/recommended.yaml` collects a set of KPIs for each metric namespace.

You can use `curl` to inspect the content of these files, e.g.:

```
curl https://observeinc.s3.us-west-2.amazonaws.com/cloudwatchmetrics/filters/recommended.yaml
```

You can host your own definition, so long as it conforms with the schema for [AWS::CloudWatch::MetricStream](https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-cloudwatch-metricstream.html). 

## Deployment

Deploy the CloudWatch Metrics Stream application using the AWS Management Console, AWS CLI, or through your CI/CD pipeline using infrastructure as code practices. Be sure to provide all necessary parameters during deployment.

## Usage

After deployment, the Firehose delivery stream will be active. It will start capturing data sent to it and automatically deliver the data to the specified S3 bucket, following the configured buffering hints.

## Monitoring

You can monitor the Firehose delivery stream through the provisioned CloudWatch Log Group. This can help you troubleshoot and understand the performance of your data streaming.

## Outputs

The stack provides no outputs.

- **Firehose**: The ARN of the Firehose delivery stream.
- **LogGroupName**: The log group used by the Firehose delivery stream.
