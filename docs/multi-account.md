# Multi-Account Deployment with StackSets

The logwriter, metricstream, and externalrole templates each have a
corresponding `-stackset` variant that deploys the app across an AWS
Organization using CloudFormation StackSets. This document covers
cross-cutting concerns common to all three.

For stack-specific parameters and behavior, see the individual docs:
- [LogWriter](logwriter.md)
- [MetricStream](metricstream.md)
- [ExternalRole](externalrole.md)

For local development and packaging workflows, see [DEVELOPER.md](../DEVELOPER.md).

## Prerequisites

1. **AWS Organizations management account** (or a delegated administrator
   account). The stackset templates use `SERVICE_MANAGED` permission model.
2. **A target Organizational Unit (OU)** containing at least one member
   account. You need the OU ID (e.g. `ou-xxxx-xxxxxxxx`).
3. **A central S3 bucket** in the management account where Firehose delivery
   streams in member accounts will write data. This bucket requires a
   cross-account bucket policy (see below).
4. **A Forwarder stack** deployed in the management account, watching the
   central bucket and forwarding data to Observe. The stacksets do not deploy
   a Forwarder.
5. **IAM capabilities**: all stackset deployments require `CAPABILITY_IAM`,
   `CAPABILITY_NAMED_IAM`, and `CAPABILITY_AUTO_EXPAND`.

## Central Bucket Permissions

The central S3 bucket must allow Firehose delivery streams from member
accounts to write objects. Without this, logwriter and metricstream instances
will fail with `Access Denied`.

The bucket policy needs to grant Firehose delivery roles in member accounts
permission to write objects and manage multipart uploads. A convenient approach
is to use a wildcard principal scoped to your organization:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "AllowFirehoseWriteFromOrg",
      "Effect": "Allow",
      "Principal": "*",
      "Action": [
        "s3:AbortMultipartUpload",
        "s3:GetBucketLocation",
        "s3:GetObject",
        "s3:ListBucket",
        "s3:ListBucketMultipartUploads",
        "s3:PutObject"
      ],
      "Resource": [
        "arn:aws:s3:::YOUR-CENTRAL-BUCKET",
        "arn:aws:s3:::YOUR-CENTRAL-BUCKET/*"
      ],
      "Condition": {
        "StringEquals": {
          "aws:PrincipalOrgID": "o-YOUR-ORG-ID"
        }
      }
    }
  ]
}
```

Replace `YOUR-CENTRAL-BUCKET` and `o-YOUR-ORG-ID` with your values. The
`aws:PrincipalOrgID` condition ensures only accounts within your organization
can write, without needing to enumerate individual account IDs.

## Operation Preferences (Concurrency)

By default, the stackset templates deploy to all accounts and regions
simultaneously. Three parameters control this behavior:

### MaxConcurrentPercentage

The percentage of target accounts to deploy to at the same time. With 200
accounts in an OU:

| Value | Behavior |
|-------|----------|
| `100` | All 200 accounts deploy simultaneously |
| `25`  | 50 at a time, in 4 waves |
| `10`  | 20 at a time, in 10 waves |

Default: **100** (maximum parallelism).

### FailureTolerancePercentage

The percentage of accounts per region that can fail before StackSets stops
deploying to remaining accounts. This does **not** roll back accounts that
already succeeded -- it only stops starting new ones.

| Value | Behavior |
|-------|----------|
| `100` | All accounts are attempted regardless of failures |
| `10`  | Stop if more than 10% of accounts fail |
| `0`   | Stop on the first failure |

Default: **100** (never abort).

### RegionConcurrencyType

When targeting multiple regions:

| Value | Behavior |
|-------|----------|
| `PARALLEL`   | Deploy to all regions at the same time |
| `SEQUENTIAL` | Finish all accounts in one region before starting the next |

Default: **PARALLEL**.

`SEQUENTIAL` is useful for canary deployments: put your canary region first in
the `TargetRegions` list, and if it fully succeeds, the remaining regions
proceed automatically.

### Choosing values

For most deployments, the defaults (100/100/PARALLEL) are appropriate -- deploy
everywhere as fast as possible. For a cautious rollout:

```
MaxConcurrentPercentage=25
FailureTolerancePercentage=10
RegionConcurrencyType=SEQUENTIAL
```

This deploys to 25% of accounts at a time in the first region, stops if more
than 10% fail, and only proceeds to the next region after the previous one
completes.

## Monitoring Operations

### List operations on a StackSet

```sh
aws cloudformation list-stack-set-operations \
  --stack-set-name STACKSET_NAME \
  --region us-west-2
```

This shows running and completed operations, including the
`OperationPreferences` that were applied.

### List instance status across accounts

```sh
aws cloudformation list-stack-instances \
  --stack-set-name STACKSET_NAME \
  --region us-west-2 \
  --query 'Summaries[*].[Account,Region,StackInstanceStatus.DetailedStatus,Status]' \
  --output table
```

Status values:
- `CURRENT` -- instance matches the latest template
- `OUTDATED` -- instance needs updating (often due to a prior failure)
- `RUNNING` -- operation in progress

### Get details on failed instances

```sh
aws cloudformation list-stack-set-operation-results \
  --stack-set-name STACKSET_NAME \
  --operation-id OPERATION_ID \
  --region us-west-2 \
  --query 'Summaries[?Status==`FAILED`].[Account,Region,StatusReason]' \
  --output table
```

## Handling Partial Failures

A partially failed StackSet has some instances in `CURRENT` (succeeded) and
others in `OUTDATED`/`FAILED`. The wrapper CloudFormation stack itself may
show `CREATE_COMPLETE` or `UPDATE_COMPLETE` if `FailureTolerancePercentage`
allowed the operation to finish.

### Retry failed instances

Fix the underlying issue (e.g. missing bucket policy, SCP blocking a region)
and update the wrapper stack. StackSets will re-attempt all `OUTDATED`
instances:

```sh
aws cloudformation update-stack \
  --stack-name WRAPPER_STACK_NAME \
  --use-previous-template \
  --parameters \
    ParameterKey=TargetOUs,UsePreviousValue=true \
    ParameterKey=TargetRegions,UsePreviousValue=true \
    ParameterKey=BucketArn,UsePreviousValue=true \
    ParameterKey=NameOverride,UsePreviousValue=true \
    ParameterKey=TemplateURL,UsePreviousValue=true \
  --capabilities CAPABILITY_IAM CAPABILITY_NAMED_IAM CAPABILITY_AUTO_EXPAND \
  --region us-west-2
```

### Revert to a previous template version

There is no built-in rollback for StackSets. To revert, update the wrapper
stack with the previous `TemplateURL`:

```sh
aws cloudformation update-stack \
  --stack-name WRAPPER_STACK_NAME \
  --use-previous-template \
  --parameters \
    ParameterKey=TemplateURL,ParameterValue=https://BUCKET.s3.REGION.amazonaws.com/aws-sam-apps/PREVIOUS_VERSION/TEMPLATE.yaml \
    ParameterKey=TargetOUs,UsePreviousValue=true \
    ParameterKey=TargetRegions,UsePreviousValue=true \
    ParameterKey=BucketArn,UsePreviousValue=true \
    ParameterKey=NameOverride,UsePreviousValue=true \
  --capabilities CAPABILITY_IAM CAPABILITY_NAMED_IAM CAPABILITY_AUTO_EXPAND \
  --region us-west-2
```

Successful instances are never rolled back by a sibling failure. If you need
all accounts on the same version, you must explicitly update or delete.

## Full Cleanup

To tear down everything -- all stack instances across all accounts and regions,
the StackSet, and the wrapper stack -- delete the wrapper stack:

```sh
aws cloudformation delete-stack \
  --stack-name WRAPPER_STACK_NAME \
  --region us-west-2
```

CloudFormation will delete all stack instances (using the template's
`OperationPreferences`, so deletions happen in parallel by default), then
delete the StackSet resource, then the wrapper stack.

Monitor progress:

```sh
aws cloudformation describe-stacks \
  --stack-name WRAPPER_STACK_NAME \
  --region us-west-2 \
  --query 'Stacks[0].StackStatus' \
  --output text
```

If the delete gets stuck, check whether any instances failed to delete:

```sh
aws cloudformation list-stack-instances \
  --stack-set-name STACKSET_NAME \
  --region us-west-2 \
  --query 'Summaries[?Status!=`CURRENT`]' \
  --output table
```

## CLI Quoting

The AWS CLI splits unquoted comma-separated values into JSON arrays. For
`CommaDelimitedList` parameters, wrap the value in quotes:

```sh
# Correct -- value is passed as the string "us-east-1,us-west-2"
'ParameterKey=TargetRegions,ParameterValue="us-east-1,us-west-2"'

# Wrong -- CLI parses this as a list, causing a validation error
ParameterKey=TargetRegions,ParameterValue=us-east-1,us-west-2
```

The same applies to `AllowedActions`, `DatastreamIds`, and any other
comma-delimited parameter. Do not backslash-escape commas -- the backslashes
end up in the parameter value.
