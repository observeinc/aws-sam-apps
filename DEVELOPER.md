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

The project's Makefile streamlines the development process with various targets:

- Build: Compile your application for deployment (make sam-build APP=<app_name>)
- Package: Package your application for AWS (make sam-package APP=<app_name>)
- Deploy: Deploy your application to AWS (make sam-deploy APP=<app_name>)
- Sync: Sync your code changes to AWS rapidly (make sam-sync APP=<app_name>)
- Validate: Check your SAM template for errors (make sam-validate APP=<app_name>)
- Publish: Share your application via AWS Serverless Application Repository (make sam-publish APP=<app_name>)
- Multi-application Commands: Build, package, or publish all applications (make sam-build-all, make sam-package-all, make sam-publish-all)
- Multi-region Commands: Manage multi-region deployments (make sam-package-all-regions)

For descriptions and usage of these targets, see the Makefile in the repository.

## Development Workflow

### Rapid Development

To mimic the production setup locally, developers can simulate the creation and use of AWS resources like S3 access points and destination URIs. This is akin to how our Terraform tests configure the environment, ensuring a seamless transition from development to production.

#### Environment Setup and Initial Deployment

First, set your AWS region and verify your identity to ensure you're operating in the correct environment:

```sh
export AWS_REGION=us-east-1
aws sts get-caller-identity
```

Next, initialize and apply the Terraform configuration. This step provisions an S3 access point and a destination URI, which are typically provided by Observe in a production setup but will be "faked" for local development:

```sh
pushd integration/modules/setup/run
terraform init
terraform plan -out tfplan
terraform apply tfplan
export ACCESS_POINT_ARM=$(terraform output -json | jq -r '.access_point.value.arn')
export S3_DESTINATION=s3://$(terraform output -json | jq -r '.access_point.value.alias')
popd
```

Afterward, create a unique S3 bucket that will act as the source for incoming data. This bucket will be configured with event notifications to trigger the necessary AWS services:

```sh
export SOURCE_BUCKET="${USER}-$(date +%s | sha256sum | head -c 8 | awk '{print tolower($0)}')"
aws s3 mb s3://$SOURCE_BUCKET --region $AWS_REGION
```

#### Simulating Forwarder Application Deployment

The `sam sync` command below simulates the `run "install_forwarder"` directive from our Terraform tests, effectively deploying the application with the necessary parameters:

```sh
pushd apps/forwarder
sam sync --stack-name app-$AWS_REGION \
    --region $AWS_REGION \
    --capabilities CAPABILITY_IAM CAPABILITY_AUTO_EXPAND CAPABILITY_NAMED_IAM \
    --parameter-overrides \
    "ParameterKey=DataAccessPointArn,ParameterValue=$ACCESS_POINT_ARM \
    ParameterKey=DestinationUri,ParameterValue=$S3_DESTINATION \
    ParameterKey=SourceBucketNames,ParameterValue='$SOURCE_BUCKET' \
    ParameterKey=MaxFileSize,ParameterValue=1"
popd
```

#### Simulating Subscriptions Setup

After the forwarder application is in place, use the AWS CLI to simulate the `run "setup_subscriptions"` step, setting up the event-driven connections that will drive the data flow in the application:

```sh
aws s3api put-bucket-notification-configuration --bucket $SOURCE_BUCKET \
    --notification-configuration '{"EventBridgeConfiguration": {}}'
```

This manual setup mirrors the automated testing environment defined in `forwarder.tftest.hcl`, allowing developers to validate the entire event flow end-to-end.

#### Cleanup

Finally, it's important to clean up the resources after testing to avoid incurring unnecessary charges:

```sh
aws cloudformation delete-stack --stack-name app-$AWS_REGION --region $AWS_REGION
watch "aws cloudformation describe-stacks --stack-name app-$AWS_REGION --region $AWS_REGION --query 'Stacks[0].StackStatus' --output text"
```

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

#### Debugging

Enable debugging mode for detailed output:

```sh
DEBUG=1 make integration-test
```

## Release

1. **Pre-release (Beta Releases on `main` branch):**
   Whenever changes are pushed to the `main` branch, our automated workflow triggers a beta release. This provides early access versions for testing and validation purposes.

2. **Full Release (Manual Trigger):**
   For creating an official release, manually trigger the release workflow from the GitHub Actions interface. This performs a full release.

3. **AWS SAM Build & Deployment:**
   - The AWS SAM application is built once at the beginning of the release phase to ensure consistency across regions.
   - AWS SAM resources are packaged and deployed across multiple AWS regions, specified in the `REGIONS` variable of our Makefile.

Upon each release, the SAM applications and their associated artifacts are packaged and uploaded to our S3 buckets. The naming convention and directory structure for these buckets are as follows:

```
observeinc-$REGION/apps/$APP/$VERSION/packaged.yaml
observeinc-$REGION/apps/$APP/latest/packaged.yaml
observeinc-$REGION/apps/$APP/beta/packaged.yaml
```

For instance:

For the collection app, version 1.0.1 being deployed to the us-west-1 region, the artifact would be located at:
observeinc-us-west-1/apps/collection/1.0.1/packaged.yaml

For the latest version of the same app in the same region:
observeinc-us-west-1/apps/collection/latest/packaged.yaml

For the beta version of the same app in the same region:
observeinc-us-west-1/apps/collection/beta/packaged.yaml

## Versioning and Contribution

We adhere to semantic versioning for our branch and commit names. For more information on the contribution process, commit message standards, and branch naming conventions, please see our [CONTRIBUTING.md](CONTRIBUTING.md).

## Additional Resources

- [AWS SAM Documentation](https://docs.aws.amazon.com/serverless-application-model/)
- [Conventional Commits](https://www.conventionalcommits.org/)
