---
name: deploy-aws-collection
description: >-
  Deploy and troubleshoot the Observe AWS Collection stack.
  Use when a customer needs help deploying a CloudFormation stack
  that collects AWS logs, metrics, and config data and sends it to Observe.
  Covers single-account stacks, parameter selection, filedrop setup,
  metric stream configuration, and common errors.
---

# Deploy Observe AWS Collection

## Agent Behavior

**Never assume customer configuration.** Before deploying, always ask the
customer for every parameter that controls what data is collected. This
includes but is not limited to:

- Which log groups to subscribe to (`LogGroupNamePrefixes` or
  `LogGroupNamePatterns`) and the specific values
- Whether to collect metrics, and if so, which mode (Filter URI vs
  Datasource-driven)
- The `NameOverride` value
- The deployment region

Only use defaults for sizing/buffering parameters (`BufferingInterval`,
`BufferingSize`) unless the customer specifies otherwise.

For credentials and filedrop values (`DestinationUri`, `DataAccessPointArn`),
ask the customer — these cannot be guessed.

## Overview

Deploys a combined AWS collection stack in a single account:
- **Forwarder** — copies S3 objects to an Observe filedrop
- **LogWriter** — subscribes CloudWatch Log Groups to a Firehose
- **MetricStream** — streams CloudWatch metrics via Firehose

Template URL pattern (production):
```
https://observeinc-{REGION}.s3.{REGION}.amazonaws.com/aws-sam-apps/latest/stack.yaml
```

## Prerequisites

Before deploying, the customer must create an **Observe Filedrop** in the Observe UI.
The filedrop provides two values needed for deployment:
- `DestinationUri` — the S3 alias URI (e.g. `s3://{alias}-s3alias/{path}/`)
- `DataAccessPointArn` — the S3 access point ARN

The IAM role name configured in the filedrop **must** match the `NameOverride`
parameter passed to the stack, or be left empty for CloudFormation to auto-generate.

## Required Parameters

| Parameter | Where to get it |
|-----------|----------------|
| `DestinationUri` | Observe UI → Connections → Filedrop → S3 destination URI |
| `DataAccessPointArn` | Observe UI → Filedrop details → access point ARN |

## CloudWatch Logs Parameters

Set **one** of these to enable log collection:

| Parameter | Format | Example |
|-----------|--------|---------|
| `LogGroupNamePrefixes` | Comma-separated prefixes | `/aws,/myapp` |
| `LogGroupNamePatterns` | Comma-separated exact patterns | `my-app-*` |

**Gotcha**: `LogGroupNamePatterns` does NOT accept regex like `.*`. Each value
must match `^(\*\|[a-zA-Z0-9-_\/]*)$`. Use `LogGroupNamePrefixes` for broad
collection (e.g. `/aws` captures all AWS service log groups).

## CloudWatch Metrics Parameters

Metrics collection has two modes:

### Mode 1: Filter URI (default, no Observe API needed)

Leave defaults — `MetricStreamFilterUri` defaults to `recommended.yaml` which
includes common AWS namespaces. However, when deployed via the `stack` template
with embedded Lambda code, the `DeployLambda` condition overrides this to
`default.yaml` (a no-op placeholder). To use this mode with the `stack`
template, the customer must also set `DatasourceID` (see Mode 2).

### Mode 2: Datasource-driven (recommended)

Create a Datasource in Observe with an `awsCollectionStackConfig` containing
the desired metric namespaces, then provide:

| Parameter | Where to get it |
|-----------|----------------|
| `DatasourceID` | Observe UI → Connections → Datasource → ID from URL |
| `GQLToken` | Observe UI → Settings → API Keys → create or copy a token |
| `ObserveAccountID` | Your Observe tenant number (e.g. `123456`) |
| `ObserveDomainName` | Your Observe domain (e.g. `observe-eng.com` or `observe-o2.com`) |
| `UpdateTimestamp` | Any changing value (e.g. current unix timestamp) to force reconfiguration |

**Important**: After an initial deploy or whenever `DatasourceID`/`GQLToken` are
changed, you must also change `UpdateTimestamp` to trigger the MetricsConfigurator
Lambda. Without this, the custom resource won't re-invoke.

## Optional Parameters

| Parameter | Purpose | Default |
|-----------|---------|---------|
| `NameOverride` | IAM role name (must match filedrop) | Auto-generated |
| `ConfigDeliveryBucketName` | Existing AWS Config S3 bucket | Empty (creates new) |
| `IncludeResourceTypes` | AWS Config resource types to collect | Empty |
| `SourceBucketNames` | Extra S3 buckets for Forwarder to read | Empty |

## Deployment Command

```bash
aws cloudformation create-stack \
  --stack-name observe-collection \
  --template-url "https://observeinc-us-west-2.s3.us-west-2.amazonaws.com/aws-sam-apps/latest/stack.yaml" \
  --parameters \
    ParameterKey=DestinationUri,ParameterValue="s3://ACCESS_POINT_ALIAS-s3alias/DATASTREAM_PATH/" \
    ParameterKey=DataAccessPointArn,ParameterValue="arn:aws:s3:REGION:ACCOUNT:accesspoint/ID" \
    ParameterKey=NameOverride,ParameterValue="observe-collection" \
    ParameterKey=LogGroupNamePrefixes,ParameterValue="/aws" \
    ParameterKey=DatasourceID,ParameterValue="DATASOURCE_ID" \
    ParameterKey=GQLToken,ParameterValue="TOKEN" \
    ParameterKey=ObserveAccountID,ParameterValue="OBSERVE_ACCOUNT_ID" \
    ParameterKey=ObserveDomainName,ParameterValue="observe-eng.com" \
    ParameterKey=UpdateTimestamp,ParameterValue="$(date +%s)" \
  --capabilities CAPABILITY_IAM CAPABILITY_NAMED_IAM CAPABILITY_AUTO_EXPAND \
  --region us-west-2
```

**All three capabilities are required.** `AUTO_EXPAND` is needed because the
template uses the SAM transform and nested stacks.

## Verifying Data Flow

### Logs

```bash
# Check the collection bucket for log data
aws s3 ls "s3://BUCKET_NAME/AWSLogs/ACCOUNT_ID/cloudwatchlogs/" --recursive | tail -5

# Check Forwarder Lambda logs for errors
aws logs get-log-events \
  --log-group-name "/aws/lambda/STACK_NAME" \
  --log-stream-name "$(aws logs describe-log-streams \
    --log-group-name '/aws/lambda/STACK_NAME' \
    --order-by LastEventTime --descending --max-items 1 \
    --query 'logStreams[0].logStreamName' --output text)" \
  --limit 10 --query 'events[].message' --output text
```

### Metrics

```bash
# Check metric stream state and filters
aws cloudwatch get-metric-stream \
  --name "STACK_NAME-MetricStream-metric-stream-0" \
  --query '{State:State,IncludeFilters:IncludeFilters[].Namespace}'

# Check for metric data in collection bucket
aws s3 ls "s3://BUCKET_NAME/AWSLogs/ACCOUNT_ID/cloudwatchmetrics/" --recursive | tail -5
```

Stack outputs provide all resource names:
```bash
aws cloudformation describe-stacks --stack-name STACK_NAME --query 'Stacks[0].Outputs'
```

---

## Troubleshooting

For detailed troubleshooting of common errors, see [troubleshooting.md](troubleshooting.md).
