# Observe AWS Config Setup

The Observe AWS Config stack automates the setup of AWS Config to capture detailed configuration information of AWS resources.

## Overview

This stack sets up AWS Config to record configuration changes and resource relationships within your AWS environment. It ensures that all supported resources are recorded and includes global resource types.

![AWS Config Setup](images/aws-config-setup.png)

## Configuration

The following parameters are required for stack configuration:

| Parameter           | Description |
|---------------------|-------------|
| `BucketName`        | The name of the S3 bucket where AWS Config stores configuration history. |
| `TopicARN`          | (Optional) The ARN of the SNS topic for AWS Config notifications. If not provided, notifications are disabled. |
| `Prefix`            | (Optional) The prefix for the S3 bucket where AWS Config stores configuration snapshots. |
| `DeliveryFrequency` | The frequency at which AWS Config delivers configuration snapshots. Options range from one hour to twenty-four hours. |

## Resources

The stack provisions the following resources:

- **ConfigurationRecorderRole**: An IAM role that allows AWS Config to assume it and grants necessary permissions for recording and delivering configurations.
- **ConfigurationRecorder**: The AWS Config recorder to track resource configurations and changes.
- **ConfigurationDeliveryChannel**: The delivery channel that defines where recorded data is stored and how often it is delivered.

## Conditions

- **DisableSNS**: Determines if SNS notifications are disabled based on the absence of the `TopicARN` parameter.

## Deployment

To deploy the Observe AWS Config stack, use the AWS SAM CLI or CloudFormation. Ensure you provide the `BucketName` parameter and optionally the `TopicARN` if you wish to receive notifications.

## Usage

Once deployed, AWS Config will start recording configurations for supported AWS resources and resource relationships. It will store the configuration history in the specified S3 bucket and deliver configuration snapshots at the specified frequency.

## Monitoring and Notifications

- **S3 Bucket**: Monitor the specified S3 bucket to ensure AWS Config writes configuration snapshots successfully.
- **SNS Topic**: If `TopicARN` is provided, subscribe to the SNS topic to receive notifications about configuration changes.

---

**Additional Notes**:

- Ensure that the S3 bucket specified in `BucketName` exists and is accessible by the stack.
- If notifications are required, the SNS topic specified in `TopicARN` should be created beforehand.
- The `Prefix` parameter can be used to organize configuration snapshots in the S3 bucket.
