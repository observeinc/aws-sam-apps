# Observe CloudWatch Metrics Stream

The Observe CloudWatch Metrics Stream application delivers CloudWatch Metrics to an Amazon S3 Bucket.

## Template Configuration

### Parameters

The application is configurable through several parameters that determine how data is buffered and delivered:

| Parameter       | Type    | Description |
|-----------------|---------|-------------|
| **`BucketArn`** | String | S3 Bucket ARN to write log records to. |
| `Prefix` | String | Optional prefix to write metrics to. |
| `FilterUri` | String | A file hosted in S3 containing list of metrics to stream. |
| `OutputFormat` | String | The output format for CloudWatch Metrics. |
| `NameOverride` | String | Set Firehose Delivery Stream name. In the absence of a value, the stack name will be used. |
| `BufferingInterval` | Number | Buffer incoming data for the specified period of time, in seconds, before delivering it to the destination.  |
| `BufferingSize` | Number | Buffer incoming data to the specified size, in MiBs, before delivering it to the destination.  |
| `ObserveAccountID` | String | The observe account id of the user.  |
| `ObserveDomainName` | String | The domain name (e.g. `observe-eng.com`) that the user is making the request from.  |
| `DatasourceID` | String | The datasource for this metric stream. If this is provided, the metric stream will not reflect the config in `MetricStreamFilterUri`, the config in `DatasourceID` will be applied instead. |
| `GQLToken` | String | The token used to retrieve metric configuration from the Observe backend.  |
| `UpdateTimestamp` | String | Unix timestamp when metric stream was created or updated.  |

### Outputs

| Output       |  Description |
|-----------------|-------------|
| FirehoseArn | Kinesis Firehose Delivery Stream ARN. CloudWatch Metric Streams subscribed to this Firehose will have their metrics batched and written to S3. |
| LogGroupName | Firehose Log Group Name. This log group will contain debugging information if Firehose fails to deliver data to S3. |

## Resources Created

The CloudWatch Metrics Stream application provisions the following AWS resources:

- **IAM Role**: Grants the Firehose service permission to access source and destination services.
- **CloudWatch Log Group**: Captures logging information from the Firehose delivery stream.
- **CloudWatch Log Stream**: A specific log stream for storing Firehose delivery logs.
- **Kinesis Firehose Delivery Stream**: The core component that manages the delivery of data to the S3 bucket.
- **CloudWatch Metrics Stream**: the component responsible for writing metrics to Kinesis Firehose.

To apply changes to the metrics via Lambda Function, you must include the `ObserveAccountID`, and `ObserveDomainName`, `DatasourceID`, `GQLToken` and `UpdateTimestamp` parameters. If these parameters are provided, the stack will also create the following:

- **Lambda Function**: Queries the datasource configuration set up in the Observe backend and updates the CloudWatch Metrics Stream accordingly.
- **Lambda IAM Role**: Grants the Lambda function permission to update the CloudWatch Metrics Stream and access the token stored in Secrets Manager.
- **Secrets Manager Secret**: Stores the token used to retrieve metric configuration from the Observe backend.

## Filtering metrics

### Via Configuration File

You may provide a URI to a pubicly readable S3 object containing a YAML or JSON definition
for what metrics to collect. Observe hosts some boilerplate filters you can use:

- `s3://observeinc/cloudwatchmetrics/filters/full.yaml` collects all metrics.
- `s3://observeinc/cloudwatchmetrics/filters/recommended.yaml` collects a set of KPIs for each metric namespace.

You can use `curl` to inspect the content of these files, e.g.:

```
curl https://observeinc.s3.us-west-2.amazonaws.com/cloudwatchmetrics/filters/recommended.yaml
```

You can host your own definition, so long as it conforms with the schema for [AWS::CloudWatch::MetricStream](https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-cloudwatch-metricstream.html). 

### Via Lambda Function

If you have selected which metrics to collect in the Observe UI while adding data, Observe will then deploy this stack with the `ObserveAccountID`, `ObserveDomainName`, `DatasourceID`, `GQLToken` and `UpdateTimestamp` parameters. This will trigger the Lambda function to query the Observe backend and update the CloudWatch Metrics Stream accordingly, without the need for a configuration file.

To replicate this behavior, you will need to retrieve a token from Observe. You can do this by making the following request to `https://{{ObserveAccountID}}.{{ObserveDomainName}}/v1/login`

```
{
    "user_email": "{{OBSERVE_USER}}",
    "user_password": "{{OBSERVE_PASSWORD}}", 
    "tokenName": "test-new-token"
}
```

You will receive a response containing a token. You can then provide this token to the stack as the `GQLToken` parameter.
