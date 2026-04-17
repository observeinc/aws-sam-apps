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
  APPS          = config configsubscription externalrole externalrole-stackset forwarder logwriter logwriter-stackset metricstream metricstream-stackset stack
  AWS_REGION    = us-west-2
  GO_BINS       = forwarder subscriber metricsconfigurator pollerconfigurator
  GO_BUILD_DIRS = bin/linux_arm64 .go/bin/linux_arm64 .go/cache .go/pkg
  TF_TESTS      = config configsubscription externalrole forwarder forwarder_s3 logwriter metricstream simple stack stack_including_metricspollerrole
  VERSION       = v1.19.2-4-gb1238b5-dirty

TARGETS:
  clean                           removes built binaries and temporary files.
  go-build                        build Go binaries.
  go-clean                        clean Go temp files.
  lambda-zips                     build Lambda ZIP archives for non-SAM-managed functions.
  ...
```

## Quick start

```
# run Go tests
→ make go-test

# package a single SAM app, will upload lambda function to S3
→ make sam-package-forwarder
→ make sam-package-logwriter
```

At this point, you will have a functional CloudFormation template under
`.aws-sam/build/regions/${AWS_REGION}/${APP}.yaml`. You can deploy this by:

- uploading manually using Cloudformation console
- deploying through `sam deploy`
- installing via Terraform

For apps with Lambda functions that SAM doesn't manage natively (logwriter,
metricstream, externalrole, stack), the packaging step automatically builds
Lambda ZIPs, uploads them to S3, and embeds the S3 references as parameter
defaults in the packaged template. An S3 bucket is auto-created using the
naming convention `aws-sam-apps-${ACCOUNT_ID}-${REGION}`. You can override
this by setting `S3_BUCKET_PREFIX`:

```
S3_BUCKET_PREFIX=my-bucket- make sam-package-logwriter
```

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

### Lambda ZIPs

Some apps (logwriter, metricstream, externalrole) use plain
`AWS::Lambda::Function` resources instead of `AWS::Serverless::Function`
because CloudFormation StackSets do not support the SAM transform.
SAM's `package` command does not handle code uploads for these. The Makefile
builds bootstrap-style ZIPs for each Lambda binary listed in `LAMBDA_ZIP_BINS`
and uploads them to S3 during `make sam-package`. The ZIPs are stored in
`.aws-sam/build/lambda-zips/`.

To build only the ZIPs (without packaging): `make lambda-zips`

### SAM package

`make sam-package` takes the artifact directory created by `sam build`, and
pushes the assets to S3. If no `S3_BUCKET_PREFIX` is provided, a default
bucket is auto-created as `aws-sam-apps-${ACCOUNT_ID}-${REGION}` using your
current AWS credentials. It will render a new CloudFormation template which
references the resulting S3 URIs. This template is stored in
`.aws-sam/build/regions/${AWS_REGION}/${APP}.yaml`

For apps with non-SAM-managed Lambdas, the packaging step also uploads the
Lambda ZIPs to S3 and post-processes the output template to embed the S3
bucket and key as parameter defaults, making the template self-contained.
The packaged template is then uploaded to S3. This is necessary because
StackSet resources require a `TemplateURL` pointing to S3 -- CloudFormation
must distribute the template to spoke accounts across the organization and
cannot reference a local file path. This upload means `sam-push` is not
needed for local StackSet testing. If a corresponding stackset template
exists (e.g. `apps/logwriter-stackset/`), it is also copied into the build
output directory and uploaded to S3.

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
make sam-package
make test-integration
```

To run a single test:

```sh
TF_TESTS="config" make test-integration
```
OR call the test target directly:
```sh
make test-integration-config
```
* Available single test targets:
```sh
make test-integration-config
make test-integration-configsubscription
make test-integration-externalrole
make test-integration-forwarder
make test-integration-forwarder_s3
make test-integration-logwriter
make test-integration-metricstream
make test-integration-simple
make test-integration-stack
make test-integration-stack_including_metricspollerrole
```

### Debugging

Enable debugging mode for detailed output:

```sh
TF_TEST_DEBUG=1 make test-integration
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
observeinc-$REGION/aws-sam-apps/$VERSION/
  forwarder.yaml
  logwriter.yaml
  metricstream.yaml
  externalrole.yaml
  stack.yaml
  logwriter-stackset.yaml
  metricstream-stackset.yaml
  externalrole-stackset.yaml
  subscriber.zip
  metricsconfigurator.zip
  pollerconfigurator.zip
  <hash>                          (forwarder code, uploaded by SAM natively)
```

Templates for apps with Lambdas (logwriter, metricstream, externalrole, stack)
have their `LambdaS3BucketPrefix` and `LambdaS3Key*` parameter defaults
pre-populated with the S3 bucket and key where the Lambda ZIPs were uploaded.

The `latest` and `beta` tags are maintained as copies of the corresponding
versioned directory.

## StackSet Templates

The `apps/` directory contains `-stackset` variants for deploying across an
AWS Organization via CloudFormation StackSets:

- `logwriter-stackset` -- deploys LogWriter across member accounts
- `metricstream-stackset` -- deploys MetricStream across member accounts
- `externalrole-stackset` -- deploys the external IAM role and PollerConfigurator

Each stackset template is a thin wrapper around `AWS::CloudFormation::StackSet`
that references the underlying app template via a `TemplateURL` parameter. The
underlying templates (logwriter, metricstream, externalrole) use plain
`AWS::Lambda::Function` (not `AWS::Serverless::Function`) because StackSets
do not support the SAM transform.

### Testing StackSets locally

The local StackSet dev workflow has three steps: package, deploy, and iterate.

#### 1. Package the app

`make sam-package-<app>` builds, packages, and uploads everything to S3 in one
command:

```sh
make sam-package-logwriter
```

This produces:
- `.aws-sam/build/regions/us-west-2/logwriter.yaml` -- packaged template with
  embedded Lambda S3 defaults
- `.aws-sam/build/regions/us-west-2/logwriter-stackset.yaml` -- stackset
  wrapper template (copied from `apps/logwriter-stackset/template.yaml`)
- Both templates and Lambda ZIPs are uploaded to the auto-created S3 bucket

#### 2. Deploy the stackset

Use `aws cloudformation create-stack` with the local stackset wrapper template.
The `TemplateURL` parameter points to the packaged template in S3:

```sh
BUCKET=aws-sam-apps-$(aws sts get-caller-identity --query Account --output text)-us-west-2
VERSION=$(git describe --tags --always --dirty)

aws cloudformation create-stack \
  --stack-name obs-logwriter-stackset \
  --template-body file://.aws-sam/build/regions/us-west-2/logwriter-stackset.yaml \
  --parameters \
    ParameterKey=TemplateURL,ParameterValue=https://${BUCKET}.s3.us-west-2.amazonaws.com/aws-sam-apps/${VERSION}/logwriter.yaml \
    ParameterKey=TargetOUs,ParameterValue=ou-XXXX-XXXXXXXX \
    ParameterKey=TargetRegions,ParameterValue=us-west-2 \
    ParameterKey=BucketArn,ParameterValue=arn:aws:s3:::YOUR-COLLECTION-BUCKET \
    ParameterKey=NameOverride,ParameterValue=obs-logwriter \
  --capabilities CAPABILITY_IAM CAPABILITY_NAMED_IAM CAPABILITY_AUTO_EXPAND \
  --region us-west-2
```

The stackset wrapper can be deployed from a local file because only the
management account reads it. The underlying template (`TemplateURL`) must be
in S3 because CloudFormation distributes it to spoke accounts.

#### 3. Iterate

After making changes to the underlying app template, re-run `make sam-package`
and update the stack:

```sh
make sam-package-logwriter

aws cloudformation update-stack \
  --stack-name obs-logwriter-stackset \
  --template-body file://.aws-sam/build/regions/us-west-2/logwriter-stackset.yaml \
  --parameters \
    ParameterKey=TemplateURL,ParameterValue=https://${BUCKET}.s3.us-west-2.amazonaws.com/aws-sam-apps/${VERSION}/logwriter.yaml \
    ParameterKey=TargetOUs,UsePreviousValue=true \
    ParameterKey=TargetRegions,UsePreviousValue=true \
    ParameterKey=BucketArn,UsePreviousValue=true \
    ParameterKey=NameOverride,UsePreviousValue=true \
  --capabilities CAPABILITY_IAM CAPABILITY_NAMED_IAM CAPABILITY_AUTO_EXPAND \
  --region us-west-2
```

If only the stackset wrapper template changed (not the underlying app), you can
update just the wrapper -- the `TemplateURL` stays the same.

#### Notes

- `NameOverride` is recommended to keep IAM role and resource names within the
  64-character limit imposed by CloudFormation StackSet naming.
- For `CommaDelimitedList` parameters (e.g. `AllowedActions`), pass commas
  literally in the value. Do not backslash-escape them -- the backslashes end
  up in the IAM policy.
- Monitor StackSet operations with
  `aws cloudformation list-stack-set-operations --stack-set-name <name>`.

## Upgrading from SAM-based templates

The logwriter, metricstream, and externalrole templates were converted from
`AWS::Serverless::Function` to plain `AWS::Lambda::Function` to support
StackSet deployments. When upgrading an existing stack to the new template
version, be aware of the following expected behaviors:

### MetricStream (filter URI path)

Customers using the filter URI path (no `DatasourceID`) will experience a
brief gap in CloudWatch Metrics collection during the stack update. The
previous CloudFormation-managed `AWS::CloudWatch::MetricStream` resource is
replaced by a Lambda-managed metric stream. CloudFormation deletes the old
resource before the Lambda custom resource creates the new one. Metrics
collection resumes automatically once the custom resource completes
(typically under one minute).

### LogWriter subscriber

The SQS event source mapping for the Subscriber Lambda changes logical IDs
during the upgrade (from SAM-generated to explicitly defined). CloudFormation
deletes the old mapping and creates a new one, causing a brief window where
SQS messages are not processed. No messages are lost -- the SQS queue retains
messages for up to 14 days, and processing resumes as soon as the new mapping
is active.

## Versioning and Contribution

We adhere to semantic versioning for our branch and commit names. For more information on the contribution process, commit message standards, and branch naming conventions, please see our [CONTRIBUTING.md](CONTRIBUTING.md).

## Additional Resources

- [AWS SAM Documentation](https://docs.aws.amazon.com/serverless-application-model/)
- [Conventional Commits](https://www.conventionalcommits.org/)
