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
5. **IAM capabilities**: all stackset deployments require `CAPABILITY_IAM`
   and `CAPABILITY_NAMED_IAM`.

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

## Quick-Create Links

Launch a stackset wrapper directly from the CloudFormation console by
clicking the badge for your management-account region below. Each link
opens the standard "Create stack" wizard prefilled with the wrapper
template — the wrapper is a regular stack that contains an
`AWS::CloudFormation::StackSet` resource, so you fill out the
`TargetOUs`, `TargetRegions`, and app-specific parameters (e.g.
`BucketArn`, `DatastreamIds`) on the parameters page.

Before clicking, make sure the [Prerequisites](#prerequisites) are in
place, especially the central bucket policy — logwriter and
metricstream instances will fail with `Access Denied` otherwise.

| Region | LogWriter | MetricStream | ExternalRole |
|--------|-----------|--------------|--------------|
| `us-west-2` | [![Static Badge](https://img.shields.io/badge/us_west_2-latest-blue?logo=amazonaws)](https://us-west-2.console.aws.amazon.com/cloudformation/home?region=us-west-2#/stacks/create/review?templateURL=https://observeinc-us-west-2.s3.amazonaws.com/aws-sam-apps/latest/logwriter-stackset.yaml) | [![Static Badge](https://img.shields.io/badge/us_west_2-latest-blue?logo=amazonaws)](https://us-west-2.console.aws.amazon.com/cloudformation/home?region=us-west-2#/stacks/create/review?templateURL=https://observeinc-us-west-2.s3.amazonaws.com/aws-sam-apps/latest/metricstream-stackset.yaml) | [![Static Badge](https://img.shields.io/badge/us_west_2-latest-blue?logo=amazonaws)](https://us-west-2.console.aws.amazon.com/cloudformation/home?region=us-west-2#/stacks/create/review?templateURL=https://observeinc-us-west-2.s3.amazonaws.com/aws-sam-apps/latest/externalrole-stackset.yaml) |
| `us-west-1` | [![Static Badge](https://img.shields.io/badge/us_west_1-latest-blue?logo=amazonaws)](https://us-west-1.console.aws.amazon.com/cloudformation/home?region=us-west-1#/stacks/create/review?templateURL=https://observeinc-us-west-1.s3.amazonaws.com/aws-sam-apps/latest/logwriter-stackset.yaml) | [![Static Badge](https://img.shields.io/badge/us_west_1-latest-blue?logo=amazonaws)](https://us-west-1.console.aws.amazon.com/cloudformation/home?region=us-west-1#/stacks/create/review?templateURL=https://observeinc-us-west-1.s3.amazonaws.com/aws-sam-apps/latest/metricstream-stackset.yaml) | [![Static Badge](https://img.shields.io/badge/us_west_1-latest-blue?logo=amazonaws)](https://us-west-1.console.aws.amazon.com/cloudformation/home?region=us-west-1#/stacks/create/review?templateURL=https://observeinc-us-west-1.s3.amazonaws.com/aws-sam-apps/latest/externalrole-stackset.yaml) |
| `us-east-2` | [![Static Badge](https://img.shields.io/badge/us_east_2-latest-blue?logo=amazonaws)](https://us-east-2.console.aws.amazon.com/cloudformation/home?region=us-east-2#/stacks/create/review?templateURL=https://observeinc-us-east-2.s3.amazonaws.com/aws-sam-apps/latest/logwriter-stackset.yaml) | [![Static Badge](https://img.shields.io/badge/us_east_2-latest-blue?logo=amazonaws)](https://us-east-2.console.aws.amazon.com/cloudformation/home?region=us-east-2#/stacks/create/review?templateURL=https://observeinc-us-east-2.s3.amazonaws.com/aws-sam-apps/latest/metricstream-stackset.yaml) | [![Static Badge](https://img.shields.io/badge/us_east_2-latest-blue?logo=amazonaws)](https://us-east-2.console.aws.amazon.com/cloudformation/home?region=us-east-2#/stacks/create/review?templateURL=https://observeinc-us-east-2.s3.amazonaws.com/aws-sam-apps/latest/externalrole-stackset.yaml) |
| `us-east-1` | [![Static Badge](https://img.shields.io/badge/us_east_1-latest-blue?logo=amazonaws)](https://us-east-1.console.aws.amazon.com/cloudformation/home?region=us-east-1#/stacks/create/review?templateURL=https://observeinc-us-east-1.s3.amazonaws.com/aws-sam-apps/latest/logwriter-stackset.yaml) | [![Static Badge](https://img.shields.io/badge/us_east_1-latest-blue?logo=amazonaws)](https://us-east-1.console.aws.amazon.com/cloudformation/home?region=us-east-1#/stacks/create/review?templateURL=https://observeinc-us-east-1.s3.amazonaws.com/aws-sam-apps/latest/metricstream-stackset.yaml) | [![Static Badge](https://img.shields.io/badge/us_east_1-latest-blue?logo=amazonaws)](https://us-east-1.console.aws.amazon.com/cloudformation/home?region=us-east-1#/stacks/create/review?templateURL=https://observeinc-us-east-1.s3.amazonaws.com/aws-sam-apps/latest/externalrole-stackset.yaml) |
| `sa-east-1` | [![Static Badge](https://img.shields.io/badge/sa_east_1-latest-blue?logo=amazonaws)](https://sa-east-1.console.aws.amazon.com/cloudformation/home?region=sa-east-1#/stacks/create/review?templateURL=https://observeinc-sa-east-1.s3.amazonaws.com/aws-sam-apps/latest/logwriter-stackset.yaml) | [![Static Badge](https://img.shields.io/badge/sa_east_1-latest-blue?logo=amazonaws)](https://sa-east-1.console.aws.amazon.com/cloudformation/home?region=sa-east-1#/stacks/create/review?templateURL=https://observeinc-sa-east-1.s3.amazonaws.com/aws-sam-apps/latest/metricstream-stackset.yaml) | [![Static Badge](https://img.shields.io/badge/sa_east_1-latest-blue?logo=amazonaws)](https://sa-east-1.console.aws.amazon.com/cloudformation/home?region=sa-east-1#/stacks/create/review?templateURL=https://observeinc-sa-east-1.s3.amazonaws.com/aws-sam-apps/latest/externalrole-stackset.yaml) |
| `eu-west-3` | [![Static Badge](https://img.shields.io/badge/eu_west_3-latest-blue?logo=amazonaws)](https://eu-west-3.console.aws.amazon.com/cloudformation/home?region=eu-west-3#/stacks/create/review?templateURL=https://observeinc-eu-west-3.s3.amazonaws.com/aws-sam-apps/latest/logwriter-stackset.yaml) | [![Static Badge](https://img.shields.io/badge/eu_west_3-latest-blue?logo=amazonaws)](https://eu-west-3.console.aws.amazon.com/cloudformation/home?region=eu-west-3#/stacks/create/review?templateURL=https://observeinc-eu-west-3.s3.amazonaws.com/aws-sam-apps/latest/metricstream-stackset.yaml) | [![Static Badge](https://img.shields.io/badge/eu_west_3-latest-blue?logo=amazonaws)](https://eu-west-3.console.aws.amazon.com/cloudformation/home?region=eu-west-3#/stacks/create/review?templateURL=https://observeinc-eu-west-3.s3.amazonaws.com/aws-sam-apps/latest/externalrole-stackset.yaml) |
| `eu-west-2` | [![Static Badge](https://img.shields.io/badge/eu_west_2-latest-blue?logo=amazonaws)](https://eu-west-2.console.aws.amazon.com/cloudformation/home?region=eu-west-2#/stacks/create/review?templateURL=https://observeinc-eu-west-2.s3.amazonaws.com/aws-sam-apps/latest/logwriter-stackset.yaml) | [![Static Badge](https://img.shields.io/badge/eu_west_2-latest-blue?logo=amazonaws)](https://eu-west-2.console.aws.amazon.com/cloudformation/home?region=eu-west-2#/stacks/create/review?templateURL=https://observeinc-eu-west-2.s3.amazonaws.com/aws-sam-apps/latest/metricstream-stackset.yaml) | [![Static Badge](https://img.shields.io/badge/eu_west_2-latest-blue?logo=amazonaws)](https://eu-west-2.console.aws.amazon.com/cloudformation/home?region=eu-west-2#/stacks/create/review?templateURL=https://observeinc-eu-west-2.s3.amazonaws.com/aws-sam-apps/latest/externalrole-stackset.yaml) |
| `eu-west-1` | [![Static Badge](https://img.shields.io/badge/eu_west_1-latest-blue?logo=amazonaws)](https://eu-west-1.console.aws.amazon.com/cloudformation/home?region=eu-west-1#/stacks/create/review?templateURL=https://observeinc-eu-west-1.s3.amazonaws.com/aws-sam-apps/latest/logwriter-stackset.yaml) | [![Static Badge](https://img.shields.io/badge/eu_west_1-latest-blue?logo=amazonaws)](https://eu-west-1.console.aws.amazon.com/cloudformation/home?region=eu-west-1#/stacks/create/review?templateURL=https://observeinc-eu-west-1.s3.amazonaws.com/aws-sam-apps/latest/metricstream-stackset.yaml) | [![Static Badge](https://img.shields.io/badge/eu_west_1-latest-blue?logo=amazonaws)](https://eu-west-1.console.aws.amazon.com/cloudformation/home?region=eu-west-1#/stacks/create/review?templateURL=https://observeinc-eu-west-1.s3.amazonaws.com/aws-sam-apps/latest/externalrole-stackset.yaml) |
| `eu-north-1` | [![Static Badge](https://img.shields.io/badge/eu_north_1-latest-blue?logo=amazonaws)](https://eu-north-1.console.aws.amazon.com/cloudformation/home?region=eu-north-1#/stacks/create/review?templateURL=https://observeinc-eu-north-1.s3.amazonaws.com/aws-sam-apps/latest/logwriter-stackset.yaml) | [![Static Badge](https://img.shields.io/badge/eu_north_1-latest-blue?logo=amazonaws)](https://eu-north-1.console.aws.amazon.com/cloudformation/home?region=eu-north-1#/stacks/create/review?templateURL=https://observeinc-eu-north-1.s3.amazonaws.com/aws-sam-apps/latest/metricstream-stackset.yaml) | [![Static Badge](https://img.shields.io/badge/eu_north_1-latest-blue?logo=amazonaws)](https://eu-north-1.console.aws.amazon.com/cloudformation/home?region=eu-north-1#/stacks/create/review?templateURL=https://observeinc-eu-north-1.s3.amazonaws.com/aws-sam-apps/latest/externalrole-stackset.yaml) |
| `eu-central-1` | [![Static Badge](https://img.shields.io/badge/eu_central_1-latest-blue?logo=amazonaws)](https://eu-central-1.console.aws.amazon.com/cloudformation/home?region=eu-central-1#/stacks/create/review?templateURL=https://observeinc-eu-central-1.s3.amazonaws.com/aws-sam-apps/latest/logwriter-stackset.yaml) | [![Static Badge](https://img.shields.io/badge/eu_central_1-latest-blue?logo=amazonaws)](https://eu-central-1.console.aws.amazon.com/cloudformation/home?region=eu-central-1#/stacks/create/review?templateURL=https://observeinc-eu-central-1.s3.amazonaws.com/aws-sam-apps/latest/metricstream-stackset.yaml) | [![Static Badge](https://img.shields.io/badge/eu_central_1-latest-blue?logo=amazonaws)](https://eu-central-1.console.aws.amazon.com/cloudformation/home?region=eu-central-1#/stacks/create/review?templateURL=https://observeinc-eu-central-1.s3.amazonaws.com/aws-sam-apps/latest/externalrole-stackset.yaml) |
| `eu-central-2` | [![Static Badge](https://img.shields.io/badge/eu_central_2-latest-blue?logo=amazonaws)](https://eu-central-2.console.aws.amazon.com/cloudformation/home?region=eu-central-2#/stacks/create/review?templateURL=https://observeinc-eu-central-2.s3.amazonaws.com/aws-sam-apps/latest/logwriter-stackset.yaml) | [![Static Badge](https://img.shields.io/badge/eu_central_2-latest-blue?logo=amazonaws)](https://eu-central-2.console.aws.amazon.com/cloudformation/home?region=eu-central-2#/stacks/create/review?templateURL=https://observeinc-eu-central-2.s3.amazonaws.com/aws-sam-apps/latest/metricstream-stackset.yaml) | [![Static Badge](https://img.shields.io/badge/eu_central_2-latest-blue?logo=amazonaws)](https://eu-central-2.console.aws.amazon.com/cloudformation/home?region=eu-central-2#/stacks/create/review?templateURL=https://observeinc-eu-central-2.s3.amazonaws.com/aws-sam-apps/latest/externalrole-stackset.yaml) |
| `ca-central-1` | [![Static Badge](https://img.shields.io/badge/ca_central_1-latest-blue?logo=amazonaws)](https://ca-central-1.console.aws.amazon.com/cloudformation/home?region=ca-central-1#/stacks/create/review?templateURL=https://observeinc-ca-central-1.s3.amazonaws.com/aws-sam-apps/latest/logwriter-stackset.yaml) | [![Static Badge](https://img.shields.io/badge/ca_central_1-latest-blue?logo=amazonaws)](https://ca-central-1.console.aws.amazon.com/cloudformation/home?region=ca-central-1#/stacks/create/review?templateURL=https://observeinc-ca-central-1.s3.amazonaws.com/aws-sam-apps/latest/metricstream-stackset.yaml) | [![Static Badge](https://img.shields.io/badge/ca_central_1-latest-blue?logo=amazonaws)](https://ca-central-1.console.aws.amazon.com/cloudformation/home?region=ca-central-1#/stacks/create/review?templateURL=https://observeinc-ca-central-1.s3.amazonaws.com/aws-sam-apps/latest/externalrole-stackset.yaml) |
| `ca-west-1` | [![Static Badge](https://img.shields.io/badge/ca_west_1-latest-blue?logo=amazonaws)](https://ca-west-1.console.aws.amazon.com/cloudformation/home?region=ca-west-1#/stacks/create/review?templateURL=https://observeinc-ca-west-1.s3.amazonaws.com/aws-sam-apps/latest/logwriter-stackset.yaml) | [![Static Badge](https://img.shields.io/badge/ca_west_1-latest-blue?logo=amazonaws)](https://ca-west-1.console.aws.amazon.com/cloudformation/home?region=ca-west-1#/stacks/create/review?templateURL=https://observeinc-ca-west-1.s3.amazonaws.com/aws-sam-apps/latest/metricstream-stackset.yaml) | [![Static Badge](https://img.shields.io/badge/ca_west_1-latest-blue?logo=amazonaws)](https://ca-west-1.console.aws.amazon.com/cloudformation/home?region=ca-west-1#/stacks/create/review?templateURL=https://observeinc-ca-west-1.s3.amazonaws.com/aws-sam-apps/latest/externalrole-stackset.yaml) |
| `ap-southeast-2` | [![Static Badge](https://img.shields.io/badge/ap_southeast_2-latest-blue?logo=amazonaws)](https://ap-southeast-2.console.aws.amazon.com/cloudformation/home?region=ap-southeast-2#/stacks/create/review?templateURL=https://observeinc-ap-southeast-2.s3.amazonaws.com/aws-sam-apps/latest/logwriter-stackset.yaml) | [![Static Badge](https://img.shields.io/badge/ap_southeast_2-latest-blue?logo=amazonaws)](https://ap-southeast-2.console.aws.amazon.com/cloudformation/home?region=ap-southeast-2#/stacks/create/review?templateURL=https://observeinc-ap-southeast-2.s3.amazonaws.com/aws-sam-apps/latest/metricstream-stackset.yaml) | [![Static Badge](https://img.shields.io/badge/ap_southeast_2-latest-blue?logo=amazonaws)](https://ap-southeast-2.console.aws.amazon.com/cloudformation/home?region=ap-southeast-2#/stacks/create/review?templateURL=https://observeinc-ap-southeast-2.s3.amazonaws.com/aws-sam-apps/latest/externalrole-stackset.yaml) |
| `ap-southeast-1` | [![Static Badge](https://img.shields.io/badge/ap_southeast_1-latest-blue?logo=amazonaws)](https://ap-southeast-1.console.aws.amazon.com/cloudformation/home?region=ap-southeast-1#/stacks/create/review?templateURL=https://observeinc-ap-southeast-1.s3.amazonaws.com/aws-sam-apps/latest/logwriter-stackset.yaml) | [![Static Badge](https://img.shields.io/badge/ap_southeast_1-latest-blue?logo=amazonaws)](https://ap-southeast-1.console.aws.amazon.com/cloudformation/home?region=ap-southeast-1#/stacks/create/review?templateURL=https://observeinc-ap-southeast-1.s3.amazonaws.com/aws-sam-apps/latest/metricstream-stackset.yaml) | [![Static Badge](https://img.shields.io/badge/ap_southeast_1-latest-blue?logo=amazonaws)](https://ap-southeast-1.console.aws.amazon.com/cloudformation/home?region=ap-southeast-1#/stacks/create/review?templateURL=https://observeinc-ap-southeast-1.s3.amazonaws.com/aws-sam-apps/latest/externalrole-stackset.yaml) |
| `ap-south-1` | [![Static Badge](https://img.shields.io/badge/ap_south_1-latest-blue?logo=amazonaws)](https://ap-south-1.console.aws.amazon.com/cloudformation/home?region=ap-south-1#/stacks/create/review?templateURL=https://observeinc-ap-south-1.s3.amazonaws.com/aws-sam-apps/latest/logwriter-stackset.yaml) | [![Static Badge](https://img.shields.io/badge/ap_south_1-latest-blue?logo=amazonaws)](https://ap-south-1.console.aws.amazon.com/cloudformation/home?region=ap-south-1#/stacks/create/review?templateURL=https://observeinc-ap-south-1.s3.amazonaws.com/aws-sam-apps/latest/metricstream-stackset.yaml) | [![Static Badge](https://img.shields.io/badge/ap_south_1-latest-blue?logo=amazonaws)](https://ap-south-1.console.aws.amazon.com/cloudformation/home?region=ap-south-1#/stacks/create/review?templateURL=https://observeinc-ap-south-1.s3.amazonaws.com/aws-sam-apps/latest/externalrole-stackset.yaml) |
| `ap-northeast-3` | [![Static Badge](https://img.shields.io/badge/ap_northeast_3-latest-blue?logo=amazonaws)](https://ap-northeast-3.console.aws.amazon.com/cloudformation/home?region=ap-northeast-3#/stacks/create/review?templateURL=https://observeinc-ap-northeast-3.s3.amazonaws.com/aws-sam-apps/latest/logwriter-stackset.yaml) | [![Static Badge](https://img.shields.io/badge/ap_northeast_3-latest-blue?logo=amazonaws)](https://ap-northeast-3.console.aws.amazon.com/cloudformation/home?region=ap-northeast-3#/stacks/create/review?templateURL=https://observeinc-ap-northeast-3.s3.amazonaws.com/aws-sam-apps/latest/metricstream-stackset.yaml) | [![Static Badge](https://img.shields.io/badge/ap_northeast_3-latest-blue?logo=amazonaws)](https://ap-northeast-3.console.aws.amazon.com/cloudformation/home?region=ap-northeast-3#/stacks/create/review?templateURL=https://observeinc-ap-northeast-3.s3.amazonaws.com/aws-sam-apps/latest/externalrole-stackset.yaml) |
| `ap-northeast-2` | [![Static Badge](https://img.shields.io/badge/ap_northeast_2-latest-blue?logo=amazonaws)](https://ap-northeast-2.console.aws.amazon.com/cloudformation/home?region=ap-northeast-2#/stacks/create/review?templateURL=https://observeinc-ap-northeast-2.s3.amazonaws.com/aws-sam-apps/latest/logwriter-stackset.yaml) | [![Static Badge](https://img.shields.io/badge/ap_northeast_2-latest-blue?logo=amazonaws)](https://ap-northeast-2.console.aws.amazon.com/cloudformation/home?region=ap-northeast-2#/stacks/create/review?templateURL=https://observeinc-ap-northeast-2.s3.amazonaws.com/aws-sam-apps/latest/metricstream-stackset.yaml) | [![Static Badge](https://img.shields.io/badge/ap_northeast_2-latest-blue?logo=amazonaws)](https://ap-northeast-2.console.aws.amazon.com/cloudformation/home?region=ap-northeast-2#/stacks/create/review?templateURL=https://observeinc-ap-northeast-2.s3.amazonaws.com/aws-sam-apps/latest/externalrole-stackset.yaml) |
| `ap-northeast-1` | [![Static Badge](https://img.shields.io/badge/ap_northeast_1-latest-blue?logo=amazonaws)](https://ap-northeast-1.console.aws.amazon.com/cloudformation/home?region=ap-northeast-1#/stacks/create/review?templateURL=https://observeinc-ap-northeast-1.s3.amazonaws.com/aws-sam-apps/latest/logwriter-stackset.yaml) | [![Static Badge](https://img.shields.io/badge/ap_northeast_1-latest-blue?logo=amazonaws)](https://ap-northeast-1.console.aws.amazon.com/cloudformation/home?region=ap-northeast-1#/stacks/create/review?templateURL=https://observeinc-ap-northeast-1.s3.amazonaws.com/aws-sam-apps/latest/metricstream-stackset.yaml) | [![Static Badge](https://img.shields.io/badge/ap_northeast_1-latest-blue?logo=amazonaws)](https://ap-northeast-1.console.aws.amazon.com/cloudformation/home?region=ap-northeast-1#/stacks/create/review?templateURL=https://observeinc-ap-northeast-1.s3.amazonaws.com/aws-sam-apps/latest/externalrole-stackset.yaml) |
| `af-south-1` | [![Static Badge](https://img.shields.io/badge/af_south_1-latest-blue?logo=amazonaws)](https://af-south-1.console.aws.amazon.com/cloudformation/home?region=af-south-1#/stacks/create/review?templateURL=https://observeinc-af-south-1.s3.amazonaws.com/aws-sam-apps/latest/logwriter-stackset.yaml) | [![Static Badge](https://img.shields.io/badge/af_south_1-latest-blue?logo=amazonaws)](https://af-south-1.console.aws.amazon.com/cloudformation/home?region=af-south-1#/stacks/create/review?templateURL=https://observeinc-af-south-1.s3.amazonaws.com/aws-sam-apps/latest/metricstream-stackset.yaml) | [![Static Badge](https://img.shields.io/badge/af_south_1-latest-blue?logo=amazonaws)](https://af-south-1.console.aws.amazon.com/cloudformation/home?region=af-south-1#/stacks/create/review?templateURL=https://observeinc-af-south-1.s3.amazonaws.com/aws-sam-apps/latest/externalrole-stackset.yaml) |
| `me-south-1` | [![Static Badge](https://img.shields.io/badge/me_south_1-latest-blue?logo=amazonaws)](https://me-south-1.console.aws.amazon.com/cloudformation/home?region=me-south-1#/stacks/create/review?templateURL=https://observeinc-me-south-1.s3.amazonaws.com/aws-sam-apps/latest/logwriter-stackset.yaml) | [![Static Badge](https://img.shields.io/badge/me_south_1-latest-blue?logo=amazonaws)](https://me-south-1.console.aws.amazon.com/cloudformation/home?region=me-south-1#/stacks/create/review?templateURL=https://observeinc-me-south-1.s3.amazonaws.com/aws-sam-apps/latest/metricstream-stackset.yaml) | [![Static Badge](https://img.shields.io/badge/me_south_1-latest-blue?logo=amazonaws)](https://me-south-1.console.aws.amazon.com/cloudformation/home?region=me-south-1#/stacks/create/review?templateURL=https://observeinc-me-south-1.s3.amazonaws.com/aws-sam-apps/latest/externalrole-stackset.yaml) |
| `me-central-1` | [![Static Badge](https://img.shields.io/badge/me_central_1-latest-blue?logo=amazonaws)](https://me-central-1.console.aws.amazon.com/cloudformation/home?region=me-central-1#/stacks/create/review?templateURL=https://observeinc-me-central-1.s3.amazonaws.com/aws-sam-apps/latest/logwriter-stackset.yaml) | [![Static Badge](https://img.shields.io/badge/me_central_1-latest-blue?logo=amazonaws)](https://me-central-1.console.aws.amazon.com/cloudformation/home?region=me-central-1#/stacks/create/review?templateURL=https://observeinc-me-central-1.s3.amazonaws.com/aws-sam-apps/latest/metricstream-stackset.yaml) | [![Static Badge](https://img.shields.io/badge/me_central_1-latest-blue?logo=amazonaws)](https://me-central-1.console.aws.amazon.com/cloudformation/home?region=me-central-1#/stacks/create/review?templateURL=https://observeinc-me-central-1.s3.amazonaws.com/aws-sam-apps/latest/externalrole-stackset.yaml) |
| `il-central-1` | [![Static Badge](https://img.shields.io/badge/il_central_1-latest-blue?logo=amazonaws)](https://il-central-1.console.aws.amazon.com/cloudformation/home?region=il-central-1#/stacks/create/review?templateURL=https://observeinc-il-central-1.s3.amazonaws.com/aws-sam-apps/latest/logwriter-stackset.yaml) | [![Static Badge](https://img.shields.io/badge/il_central_1-latest-blue?logo=amazonaws)](https://il-central-1.console.aws.amazon.com/cloudformation/home?region=il-central-1#/stacks/create/review?templateURL=https://observeinc-il-central-1.s3.amazonaws.com/aws-sam-apps/latest/metricstream-stackset.yaml) | [![Static Badge](https://img.shields.io/badge/il_central_1-latest-blue?logo=amazonaws)](https://il-central-1.console.aws.amazon.com/cloudformation/home?region=il-central-1#/stacks/create/review?templateURL=https://observeinc-il-central-1.s3.amazonaws.com/aws-sam-apps/latest/externalrole-stackset.yaml) |
| `mx-central-1` | [![Static Badge](https://img.shields.io/badge/mx_central_1-latest-blue?logo=amazonaws)](https://mx-central-1.console.aws.amazon.com/cloudformation/home?region=mx-central-1#/stacks/create/review?templateURL=https://observeinc-mx-central-1.s3.amazonaws.com/aws-sam-apps/latest/logwriter-stackset.yaml) | [![Static Badge](https://img.shields.io/badge/mx_central_1-latest-blue?logo=amazonaws)](https://mx-central-1.console.aws.amazon.com/cloudformation/home?region=mx-central-1#/stacks/create/review?templateURL=https://observeinc-mx-central-1.s3.amazonaws.com/aws-sam-apps/latest/metricstream-stackset.yaml) | [![Static Badge](https://img.shields.io/badge/mx_central_1-latest-blue?logo=amazonaws)](https://mx-central-1.console.aws.amazon.com/cloudformation/home?region=mx-central-1#/stacks/create/review?templateURL=https://observeinc-mx-central-1.s3.amazonaws.com/aws-sam-apps/latest/externalrole-stackset.yaml) |

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

> **⚠️ Silent-failure warning.** With `FailureTolerancePercentage=100`, the
> StackSet operation reports `SUCCEEDED` even if every member-account instance
> failed. The wrapper stack's `CREATE_COMPLETE`/`UPDATE_COMPLETE` only means
> "the operation ran"; it does **not** mean the stack instances are healthy.
> Always check per-instance status after deploy:
>
> - **AWS console:** open CloudFormation → **StackSets** (left sidebar) →
>   click the stackset name → **Stack instances** tab. Each row shows
>   `Status` (e.g. `CURRENT`, `OUTDATED`) and `Detailed status`
>   (e.g. `SUCCEEDED`, `FAILED`). Failed rows have a `Status reason` you can
>   click to see the underlying cause.
> - **CLI:**
>   ```sh
>   aws cloudformation list-stack-instances \
>     --stack-set-name <name> \
>     --query 'Summaries[].[Account,Region,Status,StackInstanceStatus.DetailedStatus]'
>   ```
>
> AWS requires `MaxConcurrentPercentage <= FailureTolerancePercentage + 1`,
> so lowering this value below 100 also caps parallelism. For full-fan-out
> deploys (100% concurrent), you must accept the silent-failure trade-off
> and check instance status manually.

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
  --capabilities CAPABILITY_IAM CAPABILITY_NAMED_IAM \
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
  --capabilities CAPABILITY_IAM CAPABILITY_NAMED_IAM \
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
