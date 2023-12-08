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

Before installing the Forwarder stack, set up an Observe Filedrop.

> [!IMPORTANT]
> When creating the Filedrop you should provide the ARN for an IAM Role that does not yet exist in your AWS account. The role will later be created by the Forwarder stack.
> This avoids a potential dependency cycle between Observe Filedrop, which requires the IAM role created by the Forwarder stack, and the Forwarder stack, which requires configuration details for Observe Filedrop.

Once your Observe Filedrop is created, take note of the following properties in the details page:

![Filedrop configuration](images/filedrop.png)

1. IAM Role Name: Use the name you provided during the Filedrop creation. This should be the suffix part of the role ARN.
2. S3 Access Point ARN: Noted during the Filedrop setup, it grants the Lambda permission to write to Filedrop.
3. Destination URI: The S3 URI where data will be written, typically starting with your customer ID followed by `s3alias`.

These parameters must be used to configure the Forwarder stack:

![Stack Configuration](images/forwarder-stack-configuration.png)

1. **Stack Name**: Use the IAM Role name from the Filedrop setup. If different, provide the Role name in the `NameOverride` parameter.
2. **DataAccessPointArn**: Grants the Lambda function permission to write to the Filedrop.
3. **DestinationUri**: Specifies where the Lambda function will write data.

Additionally, you may configure the following optional parameters:

- **SourceBucketNames**: Comma-delimited list of S3 Bucket names for read permissions. The wildcard pattern `*` is supported. This parameter only grants the forwarder read permissions from the provided buckets. In order to copy objects, you must trigger the lambda on object creation through a [supported subscription method](#s3-bucket-subscription).
- **ContentTypeOverrides**: Comma-delimited list of [content type overrides](#content-type-overrides).

## S3 Bucket Subscription

To forward files from an S3 bucket to the Filedrop:

1. Include the bucket name in `SourceBucketNames` or use a wildcard pattern.
2. Configure S3 Event Notifications to trigger the Forwarder's SQS queue.

**Note**: The Forwarder stack does not manage source buckets. You must manually set up the event notifications using one of the following methods:

### Subscribing an S3 bucket using EventBridge

The simplest method to configure is to enable [S3 Event Notifications with Amazon EventBridge](https://aws.amazon.com/blogs/aws/new-use-amazon-s3-event-notifications-with-amazon-eventbridge/).

To configure this method, go the `Event Notifications` section of the S3 bucket properties page:

![S3 properties](images/eb_s3_events_1.png)

And enable EventBridge events:
![Enable EventBridge](images/eb_s3_enable_1.png)

### Subscribing an S3 bucket using S3 Bucket Notifications

An S3 bucket can alternatively be configured to directly trigger the SQS queue
managed by the Forwarder. This method requires that the Forwarder have already
been successfully installed and configured with read permissions for the bucket
you wish to subscribe. This method is limited to a single destination per bucket.

To configure this method, go the `Event Notifications` section of the S3 bucket properties page:

![S3 properties](images/eb_s3_events_1.png)

Click "Create Event Notification" and provide:

- an event name
- under `Event Types`, select `All object create Events`
- under `Destination`, select `SQS Queue` and from the dropdown pick the item that has the same name as your Forwarder stack.

### Subscribing an S3 bucket using SNS

You may also consider forwarding S3 event notifications to an existing SNS topic in order to route the messages to multiple consumers.
In this case you would:
- create an SNS topic

- subscribe the SNS topic to the Forwarder SQS queue
- subscribe the S3 bucket to the SNS topic

## Message Logs

The Forwarder logs all messages it processes to Filedrop. Logs are stored in the format: `forwarder/{lambda_arn}/{request_id}`. These logs help with introspection and can forward events from AWS sources that can send messages via SQS.

## Content Type Overrides

Filedrop relies on the object content type in order to determine how to parse a file. You may encounter situations where the object content type does not accurately reflect the object contents. In such cases, you can provide a `ContentTypeOverrides` parameter which adjusts content types based on the object URI being processed.

The format for `ContentTypeOverrides` is a comma-delimited list of key value pairs. Each pair is composed of a regular expression and a content type, separated by `=`. Upon processing an object, the forwarder will match each regular expression against the object URI. Once a match is found, we will use the associated content type.

The following table lists some example uses of the `ContentTypeOverrides` parameter:

| Parameter Value                       | Description                                                                                                |
|---------------------------------------|------------------------------------------------------------------------------------------------------------|
| `.*=application/json`                 | Set `application/json` for all copied files                                                                |
| `\.csv$=text/csv,txt=text/plain`      | Set `text/csv` for all files ending in `.csv`. Otherwise, set `text/plain` for all files containing `txt`. |
| `^s3://example/=application/x-ndjson` | Set `application/x-ndjson` for all objects sourced from the `example` bucket.                              |
