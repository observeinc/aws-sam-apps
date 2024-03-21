# Observe AWS Stack

The Observe AWS stack is designed to aggregate and forward a wide range of AWS resource data, logs, and metrics to Observe, facilitating comprehensive observability of your AWS environment.

## Overview

This stack centralizes the collection process by deploying necessary AWS services and linking them to ensure seamless data flow towards your Observe Filedrop.

## Key Components

The collection stack orchestrates the following components:

- **S3 Bucket**: Stores data before it's sent to Observe.
- **SNS Topic**: Aggregates notifications from various AWS services.
- **Forwarder Application**: Routes data from S3 to Observe. Refer to [Forwarder Setup](forwarder.md) for detailed configuration.
- **Config Application**: Sets up AWS Config for resource tracking. See [Config Setup](config.md) for more information.
- **Subscriber Application**: Subscribes to log groups and forwards logs to Observe. Configuration details are available in [Subscriber Setup](subscriber.md).
- **Firehose Application**: Manages data streaming to the destination. Refer to the specific [Firehose Setup](firehose.md) for guidance.

## Configuration

The stack requires the following parameters:

| Parameter            | Description |
|----------------------|-------------|
| `DataAccessPointArn` | ARN for your Observe Filedrop access point. |
| `DestinationUri`     | URI where data will be written in the Filedrop. |
| `InstallConfig`      | Flag to install AWS Config. Set to `false` if AWS Config is already configured in the region. |

Optional parameters include patterns and prefixes for log group names, as well as bucket names and topic ARNs for data sources.

## Conditions

The stack includes conditions to manage the installation of optional components based on the provided parameters.

## Resources

The stack provisions a range of AWS resources, including an SNS topic for notifications, an S3 bucket for data storage, and applications for forwarding, configuration, and subscription, as outlined in their respective documents.

## Deployment

To deploy the Observe Stack, provide the necessary parameters such as `DataAccessPointArn` and `DestinationUri`. Use the AWS SAM CLI or CloudFormation for deployment.

## Usage

Once the stack is deployed:

1. Data from specified AWS resources is collected and stored temporarily in the S3 bucket.
2. The Forwarder application processes this data and writes it to the Filedrop.
3. AWS Config data, if enabled, is also collected and sent to the same destination.
4. CloudWatch logs are subscribed and forwarded as per the Subscriber application's configuration.

## Outputs

The stack provides outputs for the created S3 bucket and SNS topic, which can be used for further configurations or integrations.

---

**Additional Notes**:

- Ensure that the Filedrop is correctly set up before deploying this stack.
- If AWS Config is already configured in your AWS region, set `InstallConfig` to `false` to prevent conflicts.
- Configure the Forwarder, Config, and Subscriber applications as per their respective guides linked above.
