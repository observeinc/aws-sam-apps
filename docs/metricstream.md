# Observe CloudWatch Metrics Stream

The Observe CloudWatch Metrics Stream application delivers CloudWatch Metrics to an Amazon S3 Bucket.

## Configuration Parameters

The application is configurable through several parameters that determine how data is buffered and delivered:

| Parameter       | Type    | Description |
|-----------------|---------|-------------|
| **`BucketARN`** | String | S3 Bucket ARN to write log records to. |
| `Prefix` | String | Optional prefix to write metrics to. |
| `FilterURI` | String | A file hosted in S3 containing list of metrics to stream. |
| `OutputFormat` | String | The output format for CloudWatch Metrics. |
| `NameOverride` | String | Set Firehose Delivery Stream name. In the absence of a value, the stack name will be used. |
| `BufferingInterval` | Number | Buffer incoming data for the specified period of time, in seconds, before delivering it to the destination. |
| `BufferingSize` | Number | Buffer incoming data to the specified size, in MiBs, before delivering it to the destination. |

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
