# Observe CloudWatch Metrics Stream

The Observe CloudWatch Metrics Stream application delivers CloudWatch Metrics to an Amazon S3 Bucket.

## Template Configuration

### Parameters

The application is configurable through several parameters that determine how data is buffered and delivered:

| Parameter       | Type    | Description |
|-----------------|---------|-------------|
| **`BucketArn`** | String | S3 Bucket ARN to write metrics records to. |
| `Prefix` | String | Optional prefix to write metrics to. |
| `FilterUri` | String | A file hosted in S3 containing list of metrics to stream. |
| `OutputFormat` | String | The output format for CloudWatch Metrics. |
| `NameOverride` | String | Set Firehose Delivery Stream name. In the absence of a value, the stack name will be used. |
| `BufferingInterval` | Number | Buffer incoming data for the specified period of time, in seconds, before delivering it to the destination.  |
| `BufferingSize` | Number | Buffer incoming data to the specified size, in MiBs, before delivering it to the destination.  |
| `ObserveAccountID` | String | The observe account id of the user.  |
| `ObserveDomainName` | String | The domain name (e.g. `observe-eng.com`) that the user is making the request from.  |
| `DatasourceID` | String | The datasource for this metric stream. Providing this will override the `MetricStreamFilterUri`. The configuration from the datasource will be used instead. |
| `GQLToken` | String | The token used to retrieve metric configuration from the Observe backend.  |
| `UpdateTimestamp` | String | Unix timestamp when metric stream was created or updated.  |
| `LambdaS3BucketPrefix` | String | Prefix for the S3 bucket that holds the MetricsConfigurator Lambda ZIP (`{prefix}-{region}`). Published `metricstream.yaml` embeds a default. |
| `LambdaS3Key` | String | S3 key for the MetricsConfigurator Lambda ZIP. |

The template is **plain CloudFormation** (no SAM transform on this app). The MetricsConfigurator runs as a standard `AWS::Lambda::Function` with code loaded from S3.

Metrics are not collected into your pipeline until **CloudWatch metric streams** exist that send matching metrics into the Firehose delivery stream. In this design those streams are **created and updated by the MetricsConfigurator Lambda**, which CloudFormation invokes (via a custom resource) on stack create and update. There is no separate “static” metric stream defined only in CloudFormation from an S3 include; the Lambda must run for that wiring to happen when this app is deployed in the configurator-enabled mode.

**Datasource vs S3 filter file:** The Lambda does one of two things:

- **`DatasourceID` set:** It uses a **GraphQL query** against Observe to read the datasource’s metric configuration, then calls the CloudWatch API to create or update metric streams (possibly **several** streams when the filter set is large). **`GQLToken`** and related Observe parameters are required for this path.
- **`DatasourceID` empty:** It uses **`FilterUri`** only—downloads the YAML/JSON from S3 and applies it via **`PutMetricStream`** (typically a single stream).

If **both** **`DatasourceID`** and a non-empty **`FilterUri`** are supplied, the implementation **uses the datasource path only**; the S3 file is not applied for that invocation. (Leaving **`DatasourceID`** empty is how you opt into the filter-file path.)

### Outputs

| Output       |  Description |
|-----------------|-------------|
| FirehoseArn | Kinesis Firehose Delivery Stream ARN. CloudWatch Metric Streams subscribed to this Firehose will have their metrics batched and written to S3. |
| LogGroupName | Firehose Log Group Name. This log group will contain debugging information if Firehose fails to deliver data to S3. |

## Resources Created

The MetricStream template always provisions:

- **IAM Role**: Grants the Firehose service permission to access source and destination services.
- **CloudWatch Log Group / Log Stream**: Firehose delivery logging.
- **Kinesis Firehose Delivery Stream**: Delivers metrics data to the S3 bucket.
- **IAM Role for CloudWatch Metric Streams**: Lets the metric stream write to Firehose.

When the **MetricsConfigurator** path is enabled (see **Datasource vs S3 filter file** above, plus packaged Lambda ZIP parameters), the stack also creates:

- **MetricsConfigurator Lambda**: Invoked by a custom resource to create or update **CloudWatch metric streams** via the CloudWatch API (including multiple streams when needed for large filter sets). Configuration comes from the Observe datasource (**GraphQL**) when **`DatasourceID`** is set, or from the YAML/JSON at **`FilterUri`** when it is not.
- **Lambda IAM Role**: `cloudwatch:PutMetricStream` / `DeleteMetricStream`, `iam:PassRole`, and (when using a datasource) Secrets Manager access for **`GQLToken`**.
- **Secrets Manager Secret**: Created only when **`DatasourceID`** is set; stores **`GQLToken`** for the GraphQL path.
- **Custom resource**: Triggers the configurator on create/update (uses **`UpdateTimestamp`** and **`FilterUri`** as appropriate).

Static **`AWS::CloudWatch::MetricStream`** resources with **`AWS::Include`** from S3 are not used; metric streams are managed by the Lambda for compatibility with StackSets and S3-based code deployment.

## Filtering metrics

### Via Configuration File

You may provide a URI to a publicly readable S3 object containing a YAML or JSON definition
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
