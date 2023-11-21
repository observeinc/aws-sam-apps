# Developer Guide

This document serves as a guide for developers working on the `forwarder` application within the AWS Serverless Application Model (SAM) framework. The following are descriptions of the Makefile targets specific to SAM that you will use throughout the development lifecycle.

## Makefile Targets for SAM

### `sam-build`

This target compiles your application and prepares it for deployment by creating a CloudFormation template. It is the first step in deploying your application.

Usage:

```sh
APP=forwarder make sam-build
```

### `sam-package`

Packages the application, uploads it to AWS S3, and generates a new CloudFormation template. This is necessary for deployment as it creates the actual package that AWS Lambda will use.

Usage:

```sh
APP=forwarder make sam-package
```

### `sam-deploy`

This target is used to deploy your packaged application to AWS. It creates or updates the CloudFormation stack with your new application version.

Usage:

```sh
APP=forwarder make sam-deploy
```

### `sam-sync`

This target is a newer addition to the AWS SAM CLI that helps you quickly synchronize your code changes to the cloud without the need to go through the entire sam build and sam deploy cycle, which can be time-consuming. This is particularly useful during development when you want to test changes rapidly.`
Usage:

```sh
APP=forwarder make sam-sync
```

### `sam-validate`

This target validates the SAM template for your application to check for any syntax or semantic errors.

Usage:

```sh
APP=forwarder make sam-validate
```

### `sam-publish`

Publishes your application to the AWS Serverless Application Repository. It's useful for sharing your application with others or reusing it across different AWS accounts or regions.

Usage:

```sh
APP=forwarder make sam-publish
```

### `sam-package-all`

This target runs the `sam-package` target for all applications in your repository. It's useful when you want to package multiple applications at once.

Usage:

```sh
make sam-package-all
```

### `sam-publish-all`

This target runs the `sam-publish` target for all applications in your repository.

Usage:

```sh
make sam-publish-all
```

### `sam-build-all`

This target runs the `sam-build` target for each application in every specified AWS region. It's useful for multi-region deployments.

Usage:

```sh
make sam-build-all
```

### `sam-package-all-regions`

This target packages and uploads all SAM applications to S3 in multiple regions, which is an extension of `sam-package-all` for multi-region support.

Usage:

```sh
make sam-package-all-regions
```

## Additional Information

Further information on each command and development workflows will be added as necessary. Developers are encouraged to contribute to this document with any knowledge that may be beneficial to the team.

```
export AWS_REGION=us-east-1
aws sts get-caller-identity

pushd integration/modules/setup/run
terraform init
terraform plan -out tfplan
terraform apply tfplan
export ACCESS_POINT_ARM=$(terraform output -json | jq -r '.access_point.value.arn')
export S3_DESTINATION=s3://$(terraform output -json | jq -r '.access_point.value.alias')
popd

export SOURCE_BUCKET="${USER}-$(date +%s | sha256sum | head -c 8 | awk '{print tolower($0)}')"
aws s3 mb s3://$SOURCE_BUCKET --region $AWS_REGION
aws s3api put-bucket-notification-configuration --bucket $SOURCE_BUCKET \
    --notification-configuration '{
        "EventBridgeConfiguration": {}
    }'

pushd apps/forwarder
sam sync --stack-name app-v4-$AWS_REGION \
    --region $AWS_REGION \
    --capabilities CAPABILITY_IAM CAPABILITY_AUTO_EXPAND CAPABILITY_NAMED_IAM \
    --parameter-overrides \
    "ParameterKey=DataAccessPointArn,ParameterValue=$ACCESS_POINT_ARM \
    ParameterKey=DestinationUri,ParameterValue=$S3_DESTINATION \
    ParameterKey=SourceBucketNames,ParameterValue='$SOURCE_BUCKET'" \
    ParameterKey=MaxFileSize,ParameterValue=1"
```

```
aws logs tail --follow /aws/lambda/app-v4-$AWS_REGION
aws s3 cp 000.json s3://$SOURCE_BUCKET/000.json
```

```
aws cloudformation delete-stack --stack-name app-v4-$AWS_REGION --region $AWS_REGION

watch "aws cloudformation describe-stacks --stack-name app-v4-$AWS_REGION --region $AWS_REGION --query 'Stacks[0].StackStatus' --output text"
```



## Testing scribbles

### Running all tests

```shell
make integration-test
```

### Running a single test

```shell
export INTEGRATION_TEST=collection

pushd integration && \
terraform init && \
terraform test -filter=tests/$INTEGRATION_TEST.tftest.hcl -verbose && \
popd
```

or

```shell
TEST_ARGS='-filter=tests/$INTEGRATION_TEST.tftest.hcl -verbose' make integration-test
```

### Debugging

```
export INTEGRATION_TEST=collection

pushd integration && \
terraform init && \
CHECK_DEBUG_FILE=debug.sh terraform test -filter=tests/$INTEGRATION_TEST.tftest.hcl -verbose terraform init && \
popd
```

or

```shell
DEBUG=1 make integration-test
```

different terminal

```shell
cd integration
watch ls debug.sh
```

### Iterating

```shell

```
