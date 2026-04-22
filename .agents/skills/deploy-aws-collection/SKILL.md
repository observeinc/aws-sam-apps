---
name: deploy-aws-collection
description: >-
  Deploy and troubleshoot the Observe AWS Collection stack in a single AWS
  account. Use when a customer needs help deploying a CloudFormation stack
  that collects AWS logs, metrics, and config data and sends it to Observe.
  Covers parameter selection, filedrop setup, metric stream configuration,
  poller configuration, and common errors.
---

# Deploy Observe AWS Collection

## Agent Behavior

**Never assume customer configuration.** Always ask for every parameter that
controls what data is collected before deploying.

Only use defaults for sizing/buffering parameters (`BufferingInterval`,
`BufferingSize`). For credentials and filedrop values (`DestinationUri`,
`DataAccessPointArn`), always ask — these cannot be guessed.

**Never ask the customer to paste secrets (API tokens, GQL tokens, etc.)
into the chat.** Secrets must not appear in plaintext in transcripts.
Instead, provide the full command with a placeholder (e.g. `YOUR_GQL_TOKEN`)
and ask the customer to fill it in and run it themselves.

**Conversation flow:** Guide the customer through setup sequentially. Do not
ask about everything at once — complete one topic before moving to the next.

1. **Orientation** — Ask the customer what they want to set up (logs,
   metrics, or both) and briefly explain the available options:
   - Logs: CloudWatch Log Group subscription via the stack's LogWriter
   - Metrics: MetricStream (push-based) or Poller (pull-based)
   Gather the deployment region and AWS profile/account at this stage.

2. **Filedrop & identity** — Collect `DestinationUri`, `DataAccessPointArn`,
   and the expected IAM role ARN. Point the customer to **Data &
   Integrations → Datastreams → AWS → [their filedrop]** in the Observe UI.

3. **Metrics configuration** (if requested) — Walk through the chosen mode
   (MetricStream filter, Datasource-driven, custom filter YAML, or Poller).
   Finalize namespace selection, build any config files, and collect Observe
   API credentials if needed. Complete this fully before moving on.

4. **Log configuration** (if requested) — Only after metrics setup is done,
   ask which log groups to subscribe to, inform them of the 2-filter limit,
   count matching groups, and check for conflicts.

5. **Deploy** — Deploy the stack (and poller if applicable) with all
   confirmed parameters.

6. **Verify** — Check data flow and report results.

7. **Cleanup** (if deleting) — After deletion, find retained log groups and
   offer to clean them up. If a poller was created, delete it from Observe.

**`NameOverride` and filedrop role matching:** The `NameOverride` parameter
determines the Forwarder's IAM role name (`{NameOverride}-filedrop`). This
role **must** match the IAM role ARN configured in the customer's Observe
filedrop. A mismatch causes `AccessDenied` errors on the Forwarder.

- Always ask the customer what IAM role name their filedrop expects.
- If they say "default" or are unsure, ask them to check the filedrop
  configuration in the Observe UI for the expected IAM role ARN.
- Do **not** accept "default" without confirming — the filedrop almost always
  has a specific role name configured, and auto-generated names will not match.

**Log group subscription filter limits:** AWS allows a maximum of **2
subscription filters per log group**. Our LogWriter adds one filter, so log
groups that already have 2 filters will fail to subscribe. Before deploying
log collection:

1. **Inform the customer** of this limitation upfront when they request log
   monitoring.
2. **Count matching log groups** using `DescribeLogGroups` with the
   appropriate `--log-group-name-prefix` or `--log-group-name-pattern` flag
   to quickly determine how many log groups will be subscribed:

   ```bash
   aws logs describe-log-groups --log-group-name-prefix "/aws" \
     --query 'logGroups[].logGroupName' --output json \
     | python3 -c "import json,sys; print(len(json.load(sys.stdin)))"
   ```

3. **If there are fewer than 200 matching log groups**, query each one for
   existing subscription filters (use parallel threads for speed) and report
   any that already have 2 filters — these will not be subscribable:

   ```bash
   aws logs describe-subscription-filters \
     --log-group-name LOG_GROUP_NAME \
     --query 'subscriptionFilters[].filterName' --output json
   ```

   Use `concurrent.futures.ThreadPoolExecutor` with ~20 workers to query
   all groups in parallel. Report the count of groups at the 2-filter limit
   and list them by name so the customer can decide how to proceed.

**Post-deletion cleanup:** After deleting a stack, some CloudWatch Log Groups
are retained (`DeletionPolicy: Retain`). Once the stack deletion completes,
find the retained log groups and ask the customer whether they'd like them
cleaned up:

```bash
aws logs describe-log-groups \
  --log-group-name-prefix "/aws/lambda/STACK_NAME_OR_NAME_OVERRIDE" \
  --query 'logGroups[].logGroupName' --output json
```

Also check for Firehose log groups:

```bash
aws logs describe-log-groups \
  --log-group-name-prefix "/aws/firehose/STACK_NAME_OR_NAME_OVERRIDE" \
  --query 'logGroups[].logGroupName' --output json
```

List the retained log groups and let the customer choose which to delete.
Do not delete them without asking.

## Overview

Deploys a combined AWS collection stack in a single account:
- **Forwarder** — copies S3 objects to an Observe filedrop
- **LogWriter** — subscribes CloudWatch Log Groups to a Firehose
- **MetricStream** — streams CloudWatch metrics via Firehose
- **MetricsPollerRole** (optional) — nested IAM role that lets Observe poll
  CloudWatch APIs directly (deployed when `ObserveAwsAccountId` is set)

Template URL pattern (production):
```
https://observeinc-{REGION}.s3.{REGION}.amazonaws.com/aws-sam-apps/latest/stack.yaml
```

## Prerequisites

Before deploying, the customer must create an **Observe Filedrop** in the
Observe UI. The filedrop provides two values needed for deployment:
- `DestinationUri` — the S3 alias URI (e.g. `s3://{alias}-s3alias/{path}/`)
- `DataAccessPointArn` — the S3 access point ARN

The IAM role name configured in the filedrop **must** match the `NameOverride`
parameter passed to the stack, or be left empty for CloudFormation to
auto-generate.

## Required Parameters

All three values below are found in the Observe UI at **Data & Integrations →
Datastreams → AWS → [the customer's filedrop]**:

| Parameter | Where to get it |
|-----------|----------------|
| `DestinationUri` | S3 destination URI shown on the filedrop page |
| `DataAccessPointArn` | S3 access point ARN shown on the filedrop page |
| IAM Role ARN | The expected IAM role ARN shown on the filedrop page (determines `NameOverride`) |

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

Metrics collection has four modes. Pick exactly one.

### Mode 1: Filter URI (default, no Observe API needed)

Leave defaults — `MetricStreamFilterUri` defaults to `recommended.yaml` which
includes common AWS namespaces. However, when deployed via the `stack`
template with embedded Lambda code, the `DeployLambda` condition overrides
this to `default.yaml` (a no-op placeholder). To use this mode with the
`stack` template, the customer must also set `DatasourceID` (see Mode 2),
or switch to Mode 3 with a custom URI.

### Mode 2: Datasource-driven

Create a Datasource in Observe with an `awsCollectionStackConfig` containing
the desired metric namespaces, then provide:

| Parameter | Where to get it |
|-----------|----------------|
| `DatasourceID` | Observe UI → Connections → Datasource → ID from URL |
| `GQLToken` | Observe UI → Settings → API Keys → create or copy a token |
| `ObserveAccountID` | Your Observe tenant number (e.g. `123456`) |
| `ObserveDomainName` | Your Observe domain (e.g. `observe-eng.com` or `observe-o2.com`) |
| `UpdateTimestamp` | Any changing value (e.g. current unix timestamp) to force reconfiguration |

**Important**: After an initial deploy or whenever `DatasourceID`/`GQLToken`
are changed, you must also change `UpdateTimestamp` to trigger the
MetricsConfigurator Lambda. Without this, the custom resource won't re-invoke.

### Mode 3: Custom filter YAML (preferred for chat-assisted setup)

Build a YAML file with `IncludeFilters` listing the desired namespaces and
optionally specific `MetricNames`. Upload it to a publicly-readable S3
location and pass the URI as `MetricStreamFilterUri`. This avoids needing
Observe API credentials for metrics configuration.

Example filter file:

```yaml
IncludeFilters:
  - Namespace: AWS/EC2
  - Namespace: AWS/Lambda
  - Namespace: AWS/S3
    MetricNames:
      - BucketSizeBytes
      - NumberOfObjects
```

Upload it (no ACL needed if the bucket already allows public reads on the
prefix; adjust the bucket policy as needed):

```bash
aws s3 cp filter.yaml s3://BUCKET/cloudwatchmetrics/filters/custom.yaml
```

Then pass the URI at stack deploy:
`MetricStreamFilterUri=s3://BUCKET/cloudwatchmetrics/filters/custom.yaml`.

### Mode 4: Poller (pull-based)

Instead of pushing metrics via a Metric Stream, Observe can **poll**
CloudWatch APIs directly by assuming an IAM role in the customer's account.
This is useful when the customer prefers pull-based collection or needs
features like `resourceFilter` and custom dimensions.

**For single-account deployments, do NOT deploy a separate externalrole
stack.** The `stack` template already includes a nested `MetricsPollerRole`
application that deploys the externalrole automatically when
`ObserveAwsAccountId` is provided. Deploy everything as one stack.

**Required parameters on the `stack` template:**

| Parameter | Where to get it |
|-----------|----------------|
| `ObserveAwsAccountId` | The Observe AWS account ID that will assume the role (typically `723346149663`) |
| `DatastreamIds` | Observe UI → Connections → Datastream → ID |

**Optional:**

| Parameter | Default |
|-----------|---------|
| `MetricsPollerAllowedActions` | `cloudwatch:GetMetricData,cloudwatch:ListMetrics,tag:GetResources` |

The IAM role name will be `{NameOverride}-metrics-poller` (or
`{StackName}-metrics-poller` if no NameOverride).

Once the stack is deployed, the poller IAM role exists but the poller itself
must still be registered in Observe. Use the GraphQL API to create it.

#### Step 1 — Collect Observe API credentials

| Parameter | Where to get it |
|-----------|----------------|
| `ObserveAccountID` | Observe tenant number |
| `ObserveDomainName` | Observe domain (e.g. `observe-eng.com`) |
| `WorkspaceID` | Observe UI → workspace ID |
| `GQLToken` | Observe UI → Settings → API Keys |

Do not ask the customer to paste the GQL token into chat. Provide the
command below with `YOUR_GQL_TOKEN` as a placeholder and have them run it.

#### Step 2 — Ask which namespaces to poll

Use the standard namespace set from Mode 1. Each query can optionally
include `metricNames` (list of specific metrics), `dimensions` (dimension
filters), and `resourceFilter` (tag-based filtering).

#### Step 3 — Create the poller via the Observe API

```bash
OBSERVE_ACCOUNT_ID="ACCOUNT_ID"
OBSERVE_DOMAIN="observe-eng.com"
GQL_TOKEN="YOUR_GQL_TOKEN"
WORKSPACE_ID="WORKSPACE_ID"
AWS_ACCOUNT_ID="TARGET_AWS_ACCOUNT_ID"
ROLE_NAME="STACK_NAME-metrics-poller"
REGION="us-west-2"

curl -s -X POST "https://${OBSERVE_ACCOUNT_ID}.${OBSERVE_DOMAIN}/v1/meta" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ${OBSERVE_ACCOUNT_ID} ${GQL_TOKEN}" \
  -d '{
    "query": "mutation { createPoller(workspaceId: \"'${WORKSPACE_ID}'\", poller: { name: \"POLLER_NAME\", datastreamId: \"DATASTREAM_ID\", interval: \"60s\", cloudWatchMetricsConfig: { period: \"60\", delay: \"60\", region: \"'${REGION}'\", assumeRoleArn: \"arn:aws:iam::'${AWS_ACCOUNT_ID}':role/'${ROLE_NAME}'\", queries: [{namespace: \"AWS/EC2\"}, {namespace: \"AWS/RDS\"}] } }) { id name } }"
  }'
```

Record the poller ID from the response — you'll need it to update or delete
the poller later.

#### Querying poller configuration

To read back a poller's current config, use an inline fragment on the
`PollerCloudWatchMetricsConfig` union type (introspection is disabled, so
you must know the type name):

```bash
curl -s -X POST "https://${OBSERVE_ACCOUNT_ID}.${OBSERVE_DOMAIN}/v1/meta" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ${OBSERVE_ACCOUNT_ID} ${GQL_TOKEN}" \
  -d '{
    "query": "{ poller(id: \"POLLER_ID\") { id name config { ... on PollerCloudWatchMetricsConfig { queries { namespace metricNames } region assumeRoleArn period delay } } } }"
  }'
```

#### Updating a poller

Use the `updatePoller` mutation with the poller ID. The full config must be
re-sent (it replaces, not patches):

```bash
curl -s -X POST "https://${OBSERVE_ACCOUNT_ID}.${OBSERVE_DOMAIN}/v1/meta" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ${OBSERVE_ACCOUNT_ID} ${GQL_TOKEN}" \
  -d '{
    "query": "mutation { updatePoller(id: \"POLLER_ID\", poller: { name: \"POLLER_NAME\", datastreamId: \"DATASTREAM_ID\", interval: \"60s\", cloudWatchMetricsConfig: { period: \"60\", delay: \"60\", region: \"REGION\", assumeRoleArn: \"ROLE_ARN\", queries: [{namespace: \"AWS/EC2\"}] } }) { id name } }"
  }'
```

#### Deleting a poller

```bash
curl -s -X POST "https://${OBSERVE_ACCOUNT_ID}.${OBSERVE_DOMAIN}/v1/meta" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ${OBSERVE_ACCOUNT_ID} ${GQL_TOKEN}" \
  -d '{
    "query": "mutation { deletePoller(id: \"POLLER_ID\") { success } }"
  }'
```

**When tearing down a stack that has a poller**, always delete the poller
from Observe before or after deleting the CloudFormation stack. The stack
deletion only removes the IAM role — it does not remove the poller
registration in Observe. A dangling poller will generate errors on the
Observe side as it tries to assume a role that no longer exists.

## Optional Parameters

| Parameter | Purpose | Default |
|-----------|---------|---------|
| `NameOverride` | IAM role name (must match filedrop) | Auto-generated |
| `ConfigDeliveryBucketName` | Existing AWS Config S3 bucket | Empty (creates new) |
| `IncludeResourceTypes` | AWS Config resource types to collect | Empty |
| `SourceBucketNames` | Extra S3 buckets for Forwarder to read | Empty |

## Deployment Command

With MetricStream (Mode 3 custom filter YAML):

```bash
aws cloudformation create-stack \
  --stack-name observe-collection \
  --template-url "https://observeinc-us-west-2.s3.us-west-2.amazonaws.com/aws-sam-apps/latest/stack.yaml" \
  --parameters \
    ParameterKey=DestinationUri,ParameterValue="s3://ACCESS_POINT_ALIAS-s3alias/DATASTREAM_PATH/" \
    ParameterKey=DataAccessPointArn,ParameterValue="arn:aws:s3:REGION:ACCOUNT:accesspoint/ID" \
    ParameterKey=NameOverride,ParameterValue="observe-collection" \
    ParameterKey=LogGroupNamePrefixes,ParameterValue="/aws" \
    ParameterKey=MetricStreamFilterUri,ParameterValue="s3://BUCKET/cloudwatchmetrics/filters/custom.yaml" \
  --capabilities CAPABILITY_IAM CAPABILITY_NAMED_IAM CAPABILITY_AUTO_EXPAND \
  --region us-west-2
```

With Poller (includes IAM role in the same stack):

```bash
aws cloudformation create-stack \
  --stack-name observe-collection \
  --template-url "https://observeinc-us-west-2.s3.us-west-2.amazonaws.com/aws-sam-apps/latest/stack.yaml" \
  --parameters \
    ParameterKey=DestinationUri,ParameterValue="s3://ACCESS_POINT_ALIAS-s3alias/DATASTREAM_PATH/" \
    ParameterKey=DataAccessPointArn,ParameterValue="arn:aws:s3:REGION:ACCOUNT:accesspoint/ID" \
    ParameterKey=NameOverride,ParameterValue="observe-collection" \
    ParameterKey=LogGroupNamePrefixes,ParameterValue="/aws" \
    ParameterKey=ObserveAwsAccountId,ParameterValue=723346149663 \
    ParameterKey=DatastreamIds,ParameterValue=DATASTREAM_ID \
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
