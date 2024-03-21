# Observe Firehose

The Observe Firehose application is designed to streamline and automate the delivery of streaming data records to an S3 destination. It is part of the Observe ecosystem, which facilitates the collection and analysis of data for observability purposes.

## Overview

The Observe Firehose application creates an Amazon Kinesis Firehose delivery stream that captures, optionally transforms, and automatically loads streaming data into an Amazon S3 bucket.

## Configuration Parameters

| Parameter       | Type    | Description |
|-----------------|---------|-------------|
| **`BucketARN`** | String | S3 Bucket ARN to write log records to. |
| `Prefix` | String | Optional prefix to write log records to. |
| `NameOverride` | String | Set Firehose Delivery Stream name. In the absence of a value, the stack name will be used. |
| `BufferingInterval` | Number | Buffer incoming data for the specified period of time, in seconds, before delivering it to the destination.  |
| `BufferingSize` | Number | Buffer incoming data to the specified size, in MiBs, before delivering it to the destination.  |
| `WriterRoleService` | String | Optional service to create writer role for.  |

## Resources Created

The Firehose application provisions the following AWS resources:

- **IAM Role**: Grants the Firehose service permission to access source and destination services.
- **CloudWatch Log Group**: Captures logging information from the Firehose delivery stream.
- **CloudWatch Log Stream**: A specific log stream for storing Firehose delivery logs.
- **Kinesis Firehose Delivery Stream**: The core component that manages the delivery of data to the S3 bucket.

## Deployment

Deploy the Firehose application using the AWS Management Console, AWS CLI, or through your CI/CD pipeline using infrastructure as code practices. Be sure to provide all necessary parameters during deployment.

## Usage

After deployment, the Firehose delivery stream will be active. It will start capturing data sent to it and automatically deliver the data to the specified S3 bucket, following the configured buffering hints.

## Monitoring

You can monitor the Firehose delivery stream through the provisioned CloudWatch Log Group. This can help you troubleshoot and understand the performance of your data streaming.

## Outputs

The stack provides the following outputs:

- **Firehose**: The ARN of the Firehose delivery stream.
- **WriterRole**: The ARN of the writer role created if the `WriterRoleService` parameter is specified.

---

**Additional Notes:**

- Ensure the S3 bucket specified in `BucketARN` exists and has the correct permissions set.
- The `Prefix` parameter can be used to organize your data within the S3 bucket effectively.
- The buffering parameters should be tuned based on the volume and velocity of your incoming data stream.
- If the `WriterRoleService` parameter is used, ensure that the specified service has permissions to assume the created writer role.

For detailed instructions on the setup and configuration of the Firehose application, refer to the `README.md` and `DEVELOPER.md` in the main repository.
