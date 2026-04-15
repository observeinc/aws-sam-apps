---
name: deploy-aws-collection
description: >-
  Deploy and troubleshoot the Observe AWS Collection stack and stacksets.
  Use when a customer needs help deploying CloudFormation stacks or stacksets
  that collect AWS logs, metrics, and config data and send it to Observe.
  Covers single-account stacks, multi-account stacksets, parameter selection,
  filedrop setup, metric stream configuration, poller configuration, and
  common errors.
---

# Deploy Observe AWS Collection

## Overview

There are two deployment models:

### Single-account: `stack` template

Deploys a combined AWS collection stack in one account:
- **Forwarder** — copies S3 objects to an Observe filedrop
- **LogWriter** — subscribes CloudWatch Log Groups to a Firehose
- **MetricStream** — streams CloudWatch metrics via Firehose

### Multi-account: stackset templates

Deploys via AWS CloudFormation StackSets across an AWS Organization:
- **logwriter-stackset** — deploys LogWriter to member accounts
- **metricstream-stackset** — deploys MetricStream to member accounts
- **externalrole-stackset** — deploys IAM roles + optional poller config to member accounts

Stacksets require a **central S3 bucket** in the management account. A
**Forwarder** (deployed separately) reads from that bucket and forwards to
Observe.

Template URL pattern (production):
```
https://observeinc-{REGION}.s3.{REGION}.amazonaws.com/aws-sam-apps/latest/{template}.yaml
```

Where `{template}` is `stack`, `logwriter-stackset`, `metricstream-stackset`,
or `externalrole-stackset`.

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

## Stackset Deployment

### Architecture

Stacksets deploy resources to member accounts in an AWS Organization. Data flows:
1. LogWriter/MetricStream in member accounts write to a **central S3 bucket**
   in the management account via Firehose
2. A **Forwarder** in the management account reads from that bucket and
   forwards to the Observe filedrop
3. ExternalRole creates IAM roles in member accounts for Observe to assume

The Forwarder must already exist and be watching the central bucket before
deploying stacksets. Do not try to deploy it as part of the stackset flow.

### Stackset common parameters

All three stackset templates share these parameters:

| Parameter | Description |
|-----------|-------------|
| `TargetOUs` | Comma-separated OU IDs (e.g. `ou-abc123`) |
| `TargetRegions` | Comma-separated regions (e.g. `us-west-2,us-east-1`) |
| `TemplateURL` | Defaults to the matching alpha/release version; usually no override needed |
| `CallAs` | `SELF` (management account) or `DELEGATED_ADMIN` |

### logwriter-stackset

#### Parameters

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `TemplateURL` | String | Yes | (from release) | S3 URL of the packaged logwriter template |
| `TargetOUs` | CommaDelimitedList | Yes | — | OU IDs to deploy to |
| `TargetRegions` | CommaDelimitedList | Yes | — | Regions to deploy stack instances to |
| `AutoDeployEnabled` | String | No | `true` | Auto-deploy to new accounts in target OUs |
| `RetainStacksOnAccountRemoval` | String | No | `false` | Retain stacks when accounts leave target OUs |
| `CallAs` | String | No | `SELF` | `SELF` (management) or `DELEGATED_ADMIN` |
| `BucketArn` | String | Yes | — | Central S3 bucket ARN for log delivery |
| `Prefix` | String | No | (empty) | Optional S3 prefix for log records |
| `LogGroupNamePatterns` | CommaDelimitedList | No | (empty) | Substring patterns to match log groups; use `*` for all |
| `LogGroupNamePrefixes` | CommaDelimitedList | No | (empty) | Prefixes to match log groups; use `*` for all |
| `ExcludeLogGroupNamePatterns` | CommaDelimitedList | No | (empty) | Regex patterns to exclude log groups |
| `DiscoveryRate` | String | No | (empty) | EventBridge rate for periodic log group discovery (e.g. `5 minutes`) |
| `FilterName` | String | No | `observe-logs-subscription` | Subscription filter name; also used as prefix to identify managed filters |
| `FilterPattern` | String | No | (empty) | CloudWatch Logs subscription filter pattern; empty matches all |
| `NameOverride` | String | No | (stack name) | Name for Lambda function and Firehose; keep short for stacksets |
| `BufferingInterval` | Number | No | `60` | Firehose buffering time in seconds (60–900) |
| `BufferingSize` | Number | No | `1` | Firehose buffering size in MiBs (1–64) |
| `NumWorkers` | String | No | (empty) | Max concurrent workers for log group processing |
| `MemorySize` | String | No | (empty) | Lambda memory in MB |
| `Timeout` | String | No | (empty) | Lambda timeout in seconds (max 900) |
| `DebugEndpoint` | String | No | (empty) | OpenTelemetry endpoint for debug telemetry |
| `Verbosity` | String | No | (empty) | Lambda log verbosity (max 9) |

Set **one** of `LogGroupNamePatterns` or `LogGroupNamePrefixes` to enable
subscription. If neither is set, no log groups are subscribed.

#### Example

```bash
aws cloudformation create-stack \
  --stack-name obs-logwriter-stackset \
  --template-url "https://observeinc-us-west-2.s3.us-west-2.amazonaws.com/aws-sam-apps/latest/logwriter-stackset.yaml" \
  --capabilities CAPABILITY_IAM CAPABILITY_NAMED_IAM CAPABILITY_AUTO_EXPAND \
  --parameters \
    'ParameterKey=TargetOUs,ParameterValue=ou-XXXX-XXXXXXXX' \
    'ParameterKey=TargetRegions,ParameterValue="us-west-2,us-east-1"' \
    'ParameterKey=BucketArn,ParameterValue=arn:aws:s3:::CENTRAL_BUCKET' \
    'ParameterKey=LogGroupNamePrefixes,ParameterValue="/aws"' \
    'ParameterKey=NameOverride,ParameterValue=obs-logwriter' \
    'ParameterKey=DiscoveryRate,ParameterValue=5 minutes' \
    'ParameterKey=BufferingInterval,ParameterValue=60' \
    'ParameterKey=BufferingSize,ParameterValue=1'
```

### metricstream-stackset

#### Parameters

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `TemplateURL` | String | Yes | (from release) | S3 URL of the packaged metricstream template |
| `TargetOUs` | CommaDelimitedList | Yes | — | OU IDs to deploy to |
| `TargetRegions` | CommaDelimitedList | Yes | — | Regions to deploy stack instances to |
| `AutoDeployEnabled` | String | No | `true` | Auto-deploy to new accounts in target OUs |
| `RetainStacksOnAccountRemoval` | String | No | `false` | Retain stacks when accounts leave target OUs |
| `CallAs` | String | No | `SELF` | `SELF` (management) or `DELEGATED_ADMIN` |
| `BucketArn` | String | Yes | — | Central S3 bucket ARN for metric delivery |
| `Prefix` | String | No | (empty) | Optional S3 prefix for metrics |
| `FilterUri` | String | No | `s3://observeinc/cloudwatchmetrics/filters/empty.yaml` | S3 URI to a YAML/JSON metrics filter file |
| `OutputFormat` | String | No | `json` | Metric stream output format (`json`, `opentelemetry0.7`, `opentelemetry1.0`) |
| `NameOverride` | String | No | (stack name) | Firehose delivery stream name |
| `BufferingInterval` | Number | No | `60` | Firehose buffering time in seconds (60–900) |
| `BufferingSize` | Number | No | `1` | Firehose buffering size in MiBs (1–64) |
| `ObserveAccountID` | String | No | (empty) | Observe account ID (for Lambda-driven config) |
| `ObserveDomainName` | String | No | (empty) | Observe domain name (for Lambda-driven config) |
| `DatasourceID` | String | No | (empty) | Datasource ID; overrides `FilterUri` if set |
| `GQLToken` | String | No | (empty) | Token for Observe backend metric config |
| `UpdateTimestamp` | String | No | (empty) | Change to force MetricsConfigurator re-run |

**Note**: `DatasourceID` has limited functionality in stacksets because the
MetricsConfigurator Lambda must reach the Observe API from each member account.
Use `FilterUri` for reliable cross-account filtering.

#### Example

```bash
aws cloudformation create-stack \
  --stack-name obs-metricstream-stackset \
  --template-url "https://observeinc-us-west-2.s3.us-west-2.amazonaws.com/aws-sam-apps/latest/metricstream-stackset.yaml" \
  --capabilities CAPABILITY_IAM CAPABILITY_NAMED_IAM CAPABILITY_AUTO_EXPAND \
  --parameters \
    'ParameterKey=TargetOUs,ParameterValue=ou-XXXX-XXXXXXXX' \
    'ParameterKey=TargetRegions,ParameterValue="us-west-2,us-east-1"' \
    'ParameterKey=BucketArn,ParameterValue=arn:aws:s3:::CENTRAL_BUCKET' \
    'ParameterKey=FilterUri,ParameterValue=s3://observeinc/cloudwatchmetrics/filters/recommended.yaml' \
    'ParameterKey=NameOverride,ParameterValue=obs-metricstream' \
    'ParameterKey=BufferingInterval,ParameterValue=60' \
    'ParameterKey=BufferingSize,ParameterValue=1'
```

### externalrole-stackset

**Always deploy with full poller configuration.** The IAM role alone is
useless without a poller to use it. Check for an existing parameter file
(e.g. `apps/externalrole-stackset/parameters-blunderdome.json`) before
constructing parameters manually.

#### Parameters

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `TemplateURL` | String | Yes | (from release) | S3 URL of the packaged externalrole template |
| `TargetOUs` | CommaDelimitedList | Yes | — | OU IDs to deploy to |
| `TargetRegions` | CommaDelimitedList | Yes | — | Regions to deploy stack instances to |
| `AutoDeployEnabled` | String | No | `true` | Auto-deploy to new accounts in target OUs |
| `RetainStacksOnAccountRemoval` | String | No | `false` | Retain stacks when accounts leave target OUs |
| `CallAs` | String | No | `SELF` | `SELF` (management) or `DELEGATED_ADMIN` |
| `ObserveAwsAccountId` | String | Yes | — | Observe AWS account ID allowed to assume the role |
| `AllowedActions` | CommaDelimitedList | Yes | — | IAM actions Observe is allowed to execute |
| `DatastreamIds` | CommaDelimitedList | Yes | — | Datastream IDs scoping the assume-role policy |
| `NameOverride` | String | No | (stack name) | IAM role name; keep short for stacksets |
| `PrimaryRegion` | String | No | `us-west-2` | Region where the IAM role is created (IAM is global) |
| `ObserveCustomerAccountId` | String | No* | (empty) | Observe customer account ID for GQL API |
| `ObserveDomainName` | String | No* | (empty) | Observe domain (e.g. `observeinc.com`) |
| `WorkspaceID` | String | No* | (empty) | Observe workspace ID for poller creation |
| `GQLToken` | String | No* | (empty) | GQL token with poller CRUD permissions |
| `PollerConfigURI` | String | No* | (empty) | S3 URI to poller config JSON file |
| `UpdateTimestamp` | String | No | (empty) | Change to force PollerConfigurator re-run |

*Poller params are technically optional but **strongly recommended**. Without
them, only the IAM role is created — no pollers are registered. All five
(`ObserveCustomerAccountId`, `ObserveDomainName`, `WorkspaceID`, `GQLToken`,
`PollerConfigURI`) must be provided together.

`PrimaryRegion` controls where the IAM role is created (IAM is global).
Other regions skip role creation but still run the PollerConfigurator.

#### Example

```bash
aws cloudformation create-stack \
  --stack-name obs-externalrole-stackset \
  --template-url "https://observeinc-us-west-2.s3.us-west-2.amazonaws.com/aws-sam-apps/latest/externalrole-stackset.yaml" \
  --capabilities CAPABILITY_IAM CAPABILITY_NAMED_IAM CAPABILITY_AUTO_EXPAND \
  --parameters \
    'ParameterKey=TargetOUs,ParameterValue=ou-XXXX-XXXXXXXX' \
    'ParameterKey=TargetRegions,ParameterValue="us-west-2,us-east-1"' \
    'ParameterKey=ObserveAwsAccountId,ParameterValue=723346149663' \
    'ParameterKey=AllowedActions,ParameterValue="cloudwatch:GetMetricData,cloudwatch:ListMetrics,tag:GetResources"' \
    'ParameterKey=DatastreamIds,ParameterValue=DATASTREAM_ID' \
    'ParameterKey=NameOverride,ParameterValue=obs-externalrole' \
    'ParameterKey=PrimaryRegion,ParameterValue=us-west-2' \
    'ParameterKey=ObserveCustomerAccountId,ParameterValue=OBSERVE_ACCOUNT_ID' \
    'ParameterKey=ObserveDomainName,ParameterValue=observe-eng.com' \
    'ParameterKey=WorkspaceID,ParameterValue=WORKSPACE_ID' \
    'ParameterKey=GQLToken,ParameterValue=TOKEN' \
    'ParameterKey=PollerConfigURI,ParameterValue=s3://BUCKET/poller-config.json' \
    "ParameterKey=UpdateTimestamp,ParameterValue=$(date +%s)"
```

### Verifying stackset data flow

```bash
# Check central bucket for data from all accounts and regions
aws s3api list-objects-v2 \
  --bucket CENTRAL_BUCKET \
  --query 'Contents | sort_by(@, &LastModified) | [-20:].[Key, LastModified, Size]' \
  --output table

# Verify stackset instances are all CURRENT
aws cloudformation list-stack-instances \
  --stack-set-name STACKSET_NAME \
  --query 'Summaries[].{Account:Account,Region:Region,Status:Status}' \
  --output table
```

Data should include paths like:
- `AWSLogs/{ACCOUNT}/cloudwatchlogs/{REGION}/...` (logs)
- `AWSLogs/{ACCOUNT}/cloudwatchmetrics/{REGION}/...` (metrics)

### Cleaning up stacksets

Delete in dependency order — instances first, then stackset, then wrapper stack:

```bash
aws cloudformation delete-stack-instances \
  --stack-set-name STACKSET_NAME \
  --deployment-targets OrganizationalUnitIds=ou-XXXX-XXXXXXXX \
  --regions us-west-2 us-east-1 \
  --no-retain-stacks

# Poll until complete
aws cloudformation describe-stack-set-operation \
  --stack-set-name STACKSET_NAME \
  --operation-id OPERATION_ID \
  --query 'StackSetOperation.Status'

# Then delete the stackset and wrapper stack
aws cloudformation delete-stack-set --stack-set-name STACKSET_NAME
aws cloudformation delete-stack --stack-name STACKSET_NAME
```

---

## Troubleshooting

For detailed troubleshooting of common errors, see [troubleshooting.md](troubleshooting.md).
