# Developer Guide

This document details the processes and commands needed to develop, build, and deploy applications within this project.

## Setup

Before you begin, ensure that you have the following tools installed:
- AWS CLI
- SAM CLI
- Docker
- Terraform
- jq

Set up your AWS credentials and configure the default region:

```sh
export AWS_REGION=us-east-1
aws configure
```

## Makefile Targets for SAM

Our `Makefile` includes various targets to streamline the development process:

### Build

To compile your application and prepare it for deployment:

```sh
APP=forwarder make sam-build
```

### Package

To package your application and upload it to S3:

```sh
APP=forwarder make sam-package
```

### Deploy

To deploy your application to AWS:

```sh
APP=forwarder make sam-deploy
```

### Sync

For rapid development, to sync your code changes to AWS without a full deployment:

```sh
APP=forwarder make sam-sync
```

### Validate

To check your SAM template for errors:

```sh
APP=forwarder make sam-validate
```

### Publish

To share your application via AWS Serverless Application Repository:

```sh
APP=forwarder make sam-publish
```

### Multi-application Commands

To build, package, or publish all applications:

```sh
make sam-build-all
make sam-package-all
make sam-publish-all
```

### Multi-region Commands

To handle multi-region deployments:

```sh
make sam-package-all-regions
```

## Development Workflow

### Rapid Development

To streamline your development workflow and test changes quickly, follow this comprehensive script which sets up the environment and synchronizes code changes to AWS efficiently:

### Environment Setup and Initial Deployment

Set your AWS region and verify your identity:

```sh
export AWS_REGION=us-east-1
aws sts get-caller-identity
```

Initialize and apply Terraform configuration:

```sh
pushd integration/modules/setup/run
terraform init
terraform plan -out tfplan
terraform apply tfplan
export ACCESS_POINT_ARM=$(terraform output -json | jq -r '.access_point.value.arn')
export S3_DESTINATION=s3://$(terraform output -json | jq -r '.access_point.value.alias')
popd
```

Create a unique S3 bucket for source files and configure event notifications:

```sh
export SOURCE_BUCKET="${USER}-$(date +%s | sha256sum | head -c 8 | awk '{print tolower($0)}')"
aws s3 mb s3://$SOURCE_BUCKET --region $AWS_REGION
aws s3api put-bucket-notification-configuration --bucket $SOURCE_BUCKET \
    --notification-configuration '{"EventBridgeConfiguration": {}}'
```

Use `sam sync` for rapid deployment of changes:

```sh
pushd apps/forwarder
sam sync --stack-name app-v4-$AWS_REGION \
    --region $AWS_REGION \
    --capabilities CAPABILITY_IAM CAPABILITY_AUTO_EXPAND CAPABILITY_NAMED_IAM \
    --parameter-overrides \
    "ParameterKey=DataAccessPointArn,ParameterValue=$ACCESS_POINT_ARM \
    ParameterKey=DestinationUri,ParameterValue=$S3_DESTINATION \
    ParameterKey=SourceBucketNames,ParameterValue='$SOURCE_BUCKET' \
    ParameterKey=MaxFileSize,ParameterValue=1"
popd
```

### Testing Changes

Monitor logs in real-time and test file uploads:

```sh
aws logs tail --follow /aws/lambda/app-v4-$AWS_REGION
aws s3 cp 000.json s3://$SOURCE_BUCKET/000.json
```

### Teardown

Once testing is complete, remove the CloudFormation stack and resources:

```sh
aws cloudformation delete-stack --stack-name app-v4-$AWS_REGION --region $AWS_REGION
watch "aws cloudformation describe-stacks --stack-name app-v4-$AWS_REGION --region $AWS_REGION --query 'Stacks[0].StackStatus' --output text"
```

Ensure to clean up after testing to avoid incurring unnecessary costs.

### Testing

To run all integration tests:

```sh
make integration-test
```

To run a single test:

```sh
export INTEGRATION_TEST=collection
TEST_ARGS='-filter=tests/$INTEGRATION_TEST.tftest.hcl -verbose' make integration-test
```

### Debugging

Enable debugging mode for detailed output:

```sh
DEBUG=1 make integration-test
```

### Versioning

We follow semantic versioning. Please ensure your branch and commit names adhere to this convention.

## Cleanup

When you're done testing, clean up the resources:

```sh
aws cloudformation delete-stack --stack-name app-v4-$AWS_REGION --region $AWS_REGION
watch "aws cloudformation describe-stacks --stack-name app-v4-$AWS_REGION --region $AWS_REGION --query 'Stacks[0].StackStatus' --output text"
```

## Contribution

We welcome contributions! Please refer to our contribution guidelines for more information on how to submit pull requests, report issues, and suggest improvements.

## Additional Resources

- [AWS SAM Documentation](https://docs.aws.amazon.com/serverless-application-model/)
- [Conventional Commits](https://www.conventionalcommits.org/)
