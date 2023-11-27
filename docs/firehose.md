# Observe Firehose

The Observe Firehose application is designed to streamline and automate the delivery of streaming data records to an S3 destination. It is part of the Observe ecosystem, which facilitates the collection and analysis of data for observability purposes.

## Overview

The Observe Firehose application creates an Amazon Kinesis Firehose delivery stream that captures, optionally transforms, and automatically loads streaming data into an Amazon S3 bucket.

## Configuration Parameters

The application is configurable through several parameters that determine how data is buffered and delivered:

- **BucketARN**: The ARN of the S3 bucket where Firehose will deliver records.
- **Prefix**: An optional prefix path within the S3 bucket where records will be stored.
- **NameOverride**: If specified, sets the name of the Firehose Delivery Stream; otherwise, the stack name is used.
- **BufferingInterval**: The amount of time Firehose buffers incoming data before delivering it (minimum 60 seconds, maximum 900 seconds).
- **BufferingSize**: The size of the buffer, in MiBs, that Firehose accumulates before delivering data (minimum 1 MiB, maximum 64 MiBs).
- **WriterRoleService**: An optional service name to create a writer role for; if not specified, no writer role is created.

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
