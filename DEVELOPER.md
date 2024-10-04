# Developer Guide

This document details the processes and commands needed to develop, build, and deploy applications within this project.

## Prequisites

Before you begin, ensure that you have the following tools installed:
- [aws-cli](https://github.com/aws/aws-cli)
- [aws-sam-cli](https://github.com/aws/aws-sam-cli)
- [Docker](https://docs.docker.com/engine/install/)
- [Terraform](https://developer.hashicorp.com/terraform/install)

Set up your AWS credentials and configure the default region:

```sh
export AWS_REGION=us-east-1
aws configure
```

## Repository organization

The most important directories:

- `apps/` contains the SAM template definitions. This way lies Cloudformation.
- `cmd/`  contains Go entrypoints used by the different Lambda functions.
- `docs/`  contains repo documentation.
- `integration/` contains Terraform for integration testing
- `pkg/`  contains Go packages
- `vendor/` contains vendored Go dependencies.

## Makefile Targets

Our Makefile encodes all development and release workflows. To list all targets
and the most important variables, run `make help`, e.g:

```
VARIABLES:
  APPS          = config configsubscription firehose forwarder logwriter metricstream stack
  AWS_REGION    = us-west-2
  GO_BINS       = forwarder subscriber metricsconfigurator
  GO_BUILD_DIRS = bin/linux_arm64 .go/bin/linux_arm64 .go/cache .go/pkg
  TF_TESTS      = config configsubscription firehose forwarder forwarder_s3 logwriter metricstream simple stack
  VERSION       = v1.19.2-4-gb1238b5-dirty

TARGETS:
  clean                           removes built binaries and temporary files.
  go-build                        build Go binaries.
  go-clean                        clean Go temp files.
  ...
```

## Quick start

```
# run Go tests
→ make go-test

# package a single SAM app, will upload lambda function to S3
→ make sam-package-forwarder
```

At this point, you will have a functional CloudFormation template under
`.aws-sam/build/regions/${AWS_REGION}/forwarder.yaml`. You can deploy this by:

- uploading manually using Cloudformation console
- deploying through `sam deploy`
- installing via Terraform

## Running tests

This repository contains both Go code and SAM templates which can be quickly
validated locally:

`make go-test` executes Go unit tests. You can use the `GOFLAGS` environment
variable to pass in additional flags. Tests are executed within a docker
container. During development you may prefer to run `go` directly in your local
environment. A dockerized environment is provided to ensure consistency in
builds and across CI.

`make go-lint` lints Go code using [golangci-lint](https://github.com/golangci/golangci-lint).

`make sam-validate` validates SAM templates in `apps/${APP}/template.yaml`. To
run validation for a specific app, run `make sam-validate-${APP}`. This command
will also lint data according to [yamllint] and [aws-sam-cli](https://github.com/aws/aws-sam-cli).

## Packaging apps

Once your tests pass, you are ready to package a SAM application by running `make sam-package`.
You will need AWS credentials allow you to write objects to an S3 bucket.

The Makefile is wired such that running `make sam-package` will run multiple
steps as a dependency graph:
- building Lambda binaries
- running `sam build`
- running `sam package`

### Building Lambda binaries

`make go-build` is responsible for building Lambda binaries. The list of
binaries to compile are controlled through the `GO_BINS` variable. The target
architecture is set to `arm64` for compatibility with the `provided.al2` Lambda
runtime.

A build is tagged with a version. By omission, the version is derived from
git. You can override the version by setting the `RELEASE_VERSION` environment
variable.

Our build process follows [go-build-template](https://github.com/thockin/go-build-template) very
closely in order to minimize file changes that would otherwise confuse Make. 

Once build is successful, you should have a set of binaries under `bin/linux_arm64`.

### SAM build

`sam build` takes the SAM templates under `apps/${APP}/template.yaml`, and
produces a directory containing all necessary templates and artifacts for that
particular app. In our case, it does not actually build any binaries. It
invokes Make in the `CodeUri` directory specified in the template:

```
  Forwarder:
    Type: AWS::Serverless::Function
    Metadata:
      BuildMethod: makefile
    Properties:
      CodeUri: ../../bin/linux_arm64
```

We create `bin/linux_arm64/Makefile` as part of the dependencies to `make
sam-package`. The code is in `lambda.mk`, and simply copies the binary from the
Go build directory, to the temporary build directory provided through the
`${ARTIFACTS_DIR}` environment variable. The `make` target is always
`build-${ResourceName}`. By convention our resource name for a Lambda function
in the CloudFormation template is always the capitalied binary name.

### SAM package

`make sam-package` takes the artifact directory created by `sam build`, and
pushes the assets to S3. If no `S3_BUCKET_PREFIX` is provied, `samcli` will
create an S3 bucket for you. It will render a new CloudFormation template which
references the resulting S3 URIs. This template is stored in
`.aws-sam/build/regions/${AWS_REGION}/${APP}.yaml`

Lambda functions can only reference binaries stored in the same region. For
this reason, the result of running `sam package` is always region specific.
When cutting a release, we must run `sam package` for every region we wish to
support. This list of regions is maintained in the `AWS_REGIONS` variable.

### Push templates 

`make sam-push` ensures the packaged template can be used by others. The result
of `sam package` is a local template file referencing remote assets. We need
to push the template file to S3 so that it can be referenced as a URI, and we
must ensure both the templates and the assets are publicly accessible.

### Pull templates 

`make sam-pull` does the reverse of `make sam-push`. Given an
`S3_BUCKET_PREFIX`, it attempts to pull all the remote templates back to the
local build directory.

This option primarily useful for testing. For one, `make sam-pull` curls data,
and will therefore fail if the S3 objects are not publicly readable. Secondly,
our integration tests run off of local files. By allowing to pull in any
existing release version, we can run the current integration tests against
older releases, which is useful in verifying fixes.

## CI workflows

A `push` workflow is triggered on every push. It executes the following sequence:

- Run tests
    - `make go-test`
    - `make go-lint`
    - `make sam-validate`
- Upload SAM assets
    - `make sam-push`
- Run integration tests
    - `make sam-pull`
    - `make test-integration`

## Development Workflow

To mimic the production setup locally, developers can simulate the creation and use of AWS resources like S3 access points and destination URIs. This is akin to how our [Terraform tests](integration/tests/forwarder.tftest.hcl) configure the environment, ensuring a seamless transition from development to production.

### Environment Setup and Initial Deployment

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

### Simulating Forwarder Application Deployment

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

### Simulating Subscriptions Setup

After the forwarder application is in place, use the AWS CLI to simulate the `run "setup_subscriptions"` step, setting up the event-driven connections that will drive the data flow in the application:

```sh
aws s3api put-bucket-notification-configuration --bucket $SOURCE_BUCKET \
    --notification-configuration '{"EventBridgeConfiguration": {}}'
```

This manual setup mirrors the automated testing environment defined in `forwarder.tftest.hcl`, allowing developers to validate the entire event flow end-to-end.

### Cleanup

Finally, it's important to clean up the resources after testing to avoid incurring unnecessary charges:

```sh
aws cloudformation delete-stack --stack-name app-$AWS_REGION --region $AWS_REGION
watch "aws cloudformation describe-stacks --stack-name app-$AWS_REGION --region $AWS_REGION --query 'Stacks[0].StackStatus' --output text"
```

## Testing

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

Every "check" step will dump a `debug.sh` file and pause execution. Be aware this happens
during setup and teardown as well

## Release

1. **Pre-release (Beta Releases on `main` branch):**
   Whenever changes are pushed to the `main` branch, our automated workflow triggers a beta release. This provides early access versions for testing and validation purposes.

2. **Full Release (`manual Trigger):**
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
