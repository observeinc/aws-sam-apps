# Observe Forwarder Application

The Observe Forwarder is an AWS Serverless Application Model (SAM) application designed to route data to an Observe Filedrop. It can be configured to handle files from S3 buckets and ingest event streams from SQS queues, including SNS or EventBridge events.

## Overview

The Forwarder stack provisions a set of AWS resources that work together to capture data events and forward them to the specified Filedrop destination in Observe.

![Forwarder Architecture](images/forwarder.png)

## Key Components

The Forwarder stack includes:

- **SQS Queue**: Receives events from S3 buckets or SNS topics.
- **Lambda Function**: Processes messages from the SQS queue and forwards them to Filedrop.
- **IAM Role**: Grants the Lambda function permissions to write to Filedrop.
- **CloudWatch Log Group**: Captures logs from the Lambda function.
- **Dead Letter Queue**: Receives messages that fail to be processed after several attempts.
- **EventBridge Rule**: Triggers the SQS queue for `s3:ObjectCreated` events.

## Installation

Before installing the Forwarder stack, set up an Observe Filedrop. When creating the Filedrop, provide an IAM Role ARN for a role that does not yet exist; this role will be created by the Forwarder stack.

**Important Configuration Details:**

1. IAM Role Name: Use the name you provided during the Filedrop creation. This should be the suffix part of the role ARN.
2. S3 Access Point ARN: Noted during the Filedrop setup, it grants the Lambda permission to write to Filedrop.
3. Destination URI: The S3 URI where data will be written, typically starting with your customer ID followed by `s3alias`.

These properties are essential for configuring the Forwarder stack parameters correctly.

![Filedrop Configuration](images/filedrop.png)

**Stack Configuration:**

- **Stack Name**: Use the IAM Role name from the Filedrop setup. If different, provide the Role name in the `NameOverride` parameter.
- **DataAccessPointArn**: Grants the Lambda function permission to write to the Filedrop.
- **DestinationUri**: Specifies where the Lambda function will write data.

**Optional Parameters:**

- **SourceBucketNames**: Comma-delimited list of S3 Bucket names for read permissions.
- **SourceTopicArns**: Comma-delimited list of SNS Topic ARNs to receive messages from.

![Stack Configuration](images/forwarder-stack-configuration.png)

## S3 Bucket Subscription

To forward files from an S3 bucket to the Filedrop:

1. Include the bucket name in `SourceBucketNames` or use a wildcard pattern.
2. Configure S3 Event Notifications to trigger the Forwarder's SQS queue.

**Note**: The Forwarder stack does not manage source buckets. You must manually set up the event notifications using one of the following methods:

### EventBridge

Enable [S3 Event Notifications with Amazon EventBridge](https://aws.amazon.com/blogs/aws/new-use-amazon-s3-event-notifications-with-amazon-eventbridge/) on the S3 bucket properties page under `Event Notifications`.

![S3 EventBridge Configuration](images/eb_s3_events_1.png)

### Direct SQS Subscription

Configure the S3 bucket to send `Object Created` events directly to the Forwarder's SQS queue under the `Event Notifications` section.

### SNS Subscription

If using SNS, create a topic and subscribe both the S3 bucket and the Forwarder's SQS queue to this topic. Update the Forwarder stack with the SNS topic ARN in `SourceTopicArns`.

## Message Logs

The Forwarder logs all messages it processes to Filedrop. Logs are stored in the format: `forwarder/{lambda_arn}/{request_id}`. These logs help with introspection and can forward events from AWS sources that can send messages via SQS.
