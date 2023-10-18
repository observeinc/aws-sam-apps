# Observe Forwarder

The forwarder stack writes data to an Observe FileDrop. The forwarder is capable of:

- copying files from an S3 bucket to the Filedrop bucket. This method can be used to load data from processes that write to S3 (e.g. VPC Flow Logs, AWS Config, AWS CloudTrail).
- logging messages from an SQS queue to the Filedrop bucket. This capability can be used to ingest event streams such as SNS or EventBridge into Observe.

## Overview

![Forwarder](images/forwarder.png)

The forwarder stack provisions the following resources in AWS:

- an SQS Queue which receives events from any configurable source.
- a Lambda function which processes messages from the queue. Messages that are not processed successfully are returned to the queue.
- an IAM role capable of writing to a Filedrop destination. This role is assumed by the Lambda function.
- a CloudWatch Log Group where logs for the Lambda function are written to.
- an SQS [Dead Letter queue](https://docs.aws.amazon.com/AWSSimpleQueueService/latest/SQSDeveloperGuide/sqs-dead-letter-queues.html) where messages that experience multiple processing errors are sent.
- an EventBridge rule that emits a message to the source queue for every `s3:ObjectCreated` event published to EventBridge.

## Installation

In order to configure the Forwarder stack you will need to first configure an Observe Filedrop.

> [!IMPORTANT]  
> When creating the Filedrop you should provide the ARN for an IAM Role that does not yet exist in your AWS account. The role will later be created by the Forwarder stack.
> This avoids a potential dependency cycle between Observe Filedrop, which requires the IAM role created by the Forwarder stack, and the Forwarder stack, which requires configuration details for Observe Filedrop.

Once your Observe Filedrop is created, take note of the following properties in the details page:

![Filedrop configuration](images/filedrop.png)

1. The IAM Role name you provided when creating the Filedrop. The name is the trailing part of the role ARN, i.e. `arn:aws:iam::<account>:role/<name>`
2. The S3 Access Point ARN created as part of Filedrop.
3. The S3 URI to which to write data to.

These parameters must be used to configure the Forwarder stack:

![Filedrop configuration](images/forwarder-stack-configuration.png)

1. Your stack name should be set to the IAM Role name you provided when creating your Filedrop. This role will be created by the stack, and must be unique within your AWS account. If you wish to use a different name for your stack, you can instead provide the Role name in the `NameOverride` parameter.
2. `DataAccessPointArn` is used to grant permission for the lambda to write data to the Observe Filedrop Access Point.
3. `DestinationUri` determines where the Lambda function will write data to. The host portion of the URI should begin with your customer ID and have `s3alias` as a suffix.

You can additionally configure the following optional parameters:

- `SourceBucketNames` is a comma-delimited list of S3 Bucket names you wish to copy objects from. The wildcard pattern `*` is supported. This parameter only grants the forwarder read permissions from the provided buckets. In order to copy objects, you must trigger the lambda on object creation through one of three supported methods detailed in the next section.
- `SourceTopicArns` is a comma-delimited list of SNS Topic ARNs you wish to receive messages from. The wildcard pattern `*` is supported. This parameter grants the topics the ability to publish to the Forwarder stack's SQS Queue.

## S3 Bucket subscription

In order to copy files from an existing S3 Bucket to Filedrop, you must:

1. ensure the `SourceBucketNames` paramaeter either contains the name of the S3 Bucket you wish to ingest data from, or a wildcard pattern.
2. subscribe S3 Event Notifications from the source bucket to the Forwarder SQS Queue.

This last step cannot be performed by the Forwarder stack, since the source
buckets are not created and managed as part of the stack. There are three
support methods for subscribing event notifications, detailed in the next
sections.

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
- update your stack to include the created topic ARN in `SourceTopicArns`
- subscribe the SNS topic to the Forwarder SQS queue
- subscribe the S3 bucket to the SNS topic

## Message logs

The forwarder logs all processed messages to Filedrop. The messages are written to objects in Filedrop under the following path format: `forwarder/{lambda_arn}/{request_id}`.

These message logs can be used to both introspect into the Forwarder's behavior, and to forward events from AWS for any source capable of sending messages via SQS. This includes any sources that can send messages to SNS, such as AWS Config or AWS CloudFormation.
