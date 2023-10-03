# Observe Forwarder

This serverless application forwards data to an Observe FileDrop. Data is
forwarded by a lambda in two ways:

- any `s3:ObjectCreated` events trigger a copy from the source bucket to the destination bucket.
- all events read out of an SQS queue are written to a file in the destination bucket.

You can use the Observe Forwarder as a cost effective means of loading files
into Observe or for exporting event streams such as EventBridge or SNS data.

## What it does

## Subscribing an S3 bucket


## How it works

## Configuration Options

### How long does setup take?

### How do I subscribe an S3 Bucket?

### How do I subscribe an SNS topic?

### How do I subscribe EventBridge events?

## Troubleshooting 
