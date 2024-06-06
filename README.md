# AWS SAM Applications for Observe Inc

Welcome to the repository for AWS Serverless Application Model (SAM) applications used by Observe Inc. This suite of applications is designed to ingest metrics, monitoring logs, spans, traces, and events from AWS accounts into Observe for comprehensive observability.

## Overview

This repository contains multiple SAM applications, each fulfilling a specific role in the observability pipeline. For detailed documentation on each application, please refer to the `docs` folder.

## Cloudformation Quick-Create Links

| Stack | Config | Forwarder |
|------------|--------|-----------|
| [![Static Badge](https://img.shields.io/badge/ap_south_1-latest-blue?logo=amazonaws)](https://ap-south-1.console.aws.amazon.com/cloudformation/home?region=ap-south-1#/stacks/create/review?templateURL=https://observeinc-ap-south-1.s3.amazonaws.com/apps/latest/stack.yaml) | [![Static Badge](https://img.shields.io/badge/ap_south_1-latest-blue?logo=amazonaws)](https://ap-south-1.console.aws.amazon.com/cloudformation/home?region=ap-south-1#/stacks/create/review?templateURL=https://observeinc-ap-south-1.s3.amazonaws.com/apps/latest/config/packaged.yaml) | [![Static Badge](https://img.shields.io/badge/ap_south_1-latest-blue?logo=amazonaws)](https://ap-south-1.console.aws.amazon.com/cloudformation/home?region=ap-south-1#/stacks/create/review?templateURL=https://observeinc-ap-south-1.s3.amazonaws.com/apps/latest/forwarder/packaged.yaml) |
| [![Static Badge](https://img.shields.io/badge/eu_north_1-latest-blue?logo=amazonaws)](https://eu-north-1.console.aws.amazon.com/cloudformation/home?region=eu-north-1#/stacks/create/review?templateURL=https://observeinc-eu-north-1.s3.amazonaws.com/apps/latest/stack.yaml) | [![Static Badge](https://img.shields.io/badge/eu_north_1-latest-blue?logo=amazonaws)](https://eu-north-1.console.aws.amazon.com/cloudformation/home?region=eu-north-1#/stacks/create/review?templateURL=https://observeinc-eu-north-1.s3.amazonaws.com/apps/latest/config/packaged.yaml) | [![Static Badge](https://img.shields.io/badge/eu_north_1-latest-blue?logo=amazonaws)](https://eu-north-1.console.aws.amazon.com/cloudformation/home?region=eu-north-1#/stacks/create/review?templateURL=https://observeinc-eu-north-1.s3.amazonaws.com/apps/latest/forwarder/packaged.yaml) |
| [![Static Badge](https://img.shields.io/badge/eu_west_3-latest-blue?logo=amazonaws)](https://eu-west-3.console.aws.amazon.com/cloudformation/home?region=eu-west-3#/stacks/create/review?templateURL=https://observeinc-eu-west-3.s3.amazonaws.com/apps/latest/stack.yaml) | [![Static Badge](https://img.shields.io/badge/eu_west_3-latest-blue?logo=amazonaws)](https://eu-west-3.console.aws.amazon.com/cloudformation/home?region=eu-west-3#/stacks/create/review?templateURL=https://observeinc-eu-west-3.s3.amazonaws.com/apps/latest/config/packaged.yaml) | [![Static Badge](https://img.shields.io/badge/eu_west_3-latest-blue?logo=amazonaws)](https://eu-west-3.console.aws.amazon.com/cloudformation/home?region=eu-west-3#/stacks/create/review?templateURL=https://observeinc-eu-west-3.s3.amazonaws.com/apps/latest/forwarder/packaged.yaml) |
| [![Static Badge](https://img.shields.io/badge/eu_west_2-latest-blue?logo=amazonaws)](https://eu-west-2.console.aws.amazon.com/cloudformation/home?region=eu-west-2#/stacks/create/review?templateURL=https://observeinc-eu-west-2.s3.amazonaws.com/apps/latest/stack.yaml) | [![Static Badge](https://img.shields.io/badge/eu_west_2-latest-blue?logo=amazonaws)](https://eu-west-2.console.aws.amazon.com/cloudformation/home?region=eu-west-2#/stacks/create/review?templateURL=https://observeinc-eu-west-2.s3.amazonaws.com/apps/latest/config/packaged.yaml) | [![Static Badge](https://img.shields.io/badge/eu_west_2-latest-blue?logo=amazonaws)](https://eu-west-2.console.aws.amazon.com/cloudformation/home?region=eu-west-2#/stacks/create/review?templateURL=https://observeinc-eu-west-2.s3.amazonaws.com/apps/latest/forwarder/packaged.yaml) |
| [![Static Badge](https://img.shields.io/badge/eu_west_1-latest-blue?logo=amazonaws)](https://eu-west-1.console.aws.amazon.com/cloudformation/home?region=eu-west-1#/stacks/create/review?templateURL=https://observeinc-eu-west-1.s3.amazonaws.com/apps/latest/stack.yaml) | [![Static Badge](https://img.shields.io/badge/eu_west_1-latest-blue?logo=amazonaws)](https://eu-west-1.console.aws.amazon.com/cloudformation/home?region=eu-west-1#/stacks/create/review?templateURL=https://observeinc-eu-west-1.s3.amazonaws.com/apps/latest/config/packaged.yaml) | [![Static Badge](https://img.shields.io/badge/eu_west_1-latest-blue?logo=amazonaws)](https://eu-west-1.console.aws.amazon.com/cloudformation/home?region=eu-west-1#/stacks/create/review?templateURL=https://observeinc-eu-west-1.s3.amazonaws.com/apps/latest/forwarder/packaged.yaml) |
| [![Static Badge](https://img.shields.io/badge/ap_northeast_3-latest-blue?logo=amazonaws)](https://ap-northeast-3.console.aws.amazon.com/cloudformation/home?region=ap-northeast-3#/stacks/create/review?templateURL=https://observeinc-ap-northeast-3.s3.amazonaws.com/apps/latest/stack.yaml) | [![Static Badge](https://img.shields.io/badge/ap_northeast_3-latest-blue?logo=amazonaws)](https://ap-northeast-3.console.aws.amazon.com/cloudformation/home?region=ap-northeast-3#/stacks/create/review?templateURL=https://observeinc-ap-northeast-3.s3.amazonaws.com/apps/latest/config/packaged.yaml) | [![Static Badge](https://img.shields.io/badge/ap_northeast_3-latest-blue?logo=amazonaws)](https://ap-northeast-3.console.aws.amazon.com/cloudformation/home?region=ap-northeast-3#/stacks/create/review?templateURL=https://observeinc-ap-northeast-3.s3.amazonaws.com/apps/latest/forwarder/packaged.yaml) |
| [![Static Badge](https://img.shields.io/badge/ap_northeast_2-latest-blue?logo=amazonaws)](https://ap-northeast-2.console.aws.amazon.com/cloudformation/home?region=ap-northeast-2#/stacks/create/review?templateURL=https://observeinc-ap-northeast-2.s3.amazonaws.com/apps/latest/stack.yaml) | [![Static Badge](https://img.shields.io/badge/ap_northeast_2-latest-blue?logo=amazonaws)](https://ap-northeast-2.console.aws.amazon.com/cloudformation/home?region=ap-northeast-2#/stacks/create/review?templateURL=https://observeinc-ap-northeast-2.s3.amazonaws.com/apps/latest/config/packaged.yaml) | [![Static Badge](https://img.shields.io/badge/ap_northeast_2-latest-blue?logo=amazonaws)](https://ap-northeast-2.console.aws.amazon.com/cloudformation/home?region=ap-northeast-2#/stacks/create/review?templateURL=https://observeinc-ap-northeast-2.s3.amazonaws.com/apps/latest/forwarder/packaged.yaml) |
| [![Static Badge](https://img.shields.io/badge/ap_northeast_1-latest-blue?logo=amazonaws)](https://ap-northeast-1.console.aws.amazon.com/cloudformation/home?region=ap-northeast-1#/stacks/create/review?templateURL=https://observeinc-ap-northeast-1.s3.amazonaws.com/apps/latest/stack.yaml) | [![Static Badge](https://img.shields.io/badge/ap_northeast_1-latest-blue?logo=amazonaws)](https://ap-northeast-1.console.aws.amazon.com/cloudformation/home?region=ap-northeast-1#/stacks/create/review?templateURL=https://observeinc-ap-northeast-1.s3.amazonaws.com/apps/latest/config/packaged.yaml) | [![Static Badge](https://img.shields.io/badge/ap_northeast_1-latest-blue?logo=amazonaws)](https://ap-northeast-1.console.aws.amazon.com/cloudformation/home?region=ap-northeast-1#/stacks/create/review?templateURL=https://observeinc-ap-northeast-1.s3.amazonaws.com/apps/latest/forwarder/packaged.yaml) |
| [![Static Badge](https://img.shields.io/badge/ca_central_1-latest-blue?logo=amazonaws)](https://ca-central-1.console.aws.amazon.com/cloudformation/home?region=ca-central-1#/stacks/create/review?templateURL=https://observeinc-ca-central-1.s3.amazonaws.com/apps/latest/stack.yaml) | [![Static Badge](https://img.shields.io/badge/ca_central_1-latest-blue?logo=amazonaws)](https://ca-central-1.console.aws.amazon.com/cloudformation/home?region=ca-central-1#/stacks/create/review?templateURL=https://observeinc-ca-central-1.s3.amazonaws.com/apps/latest/config/packaged.yaml) | [![Static Badge](https://img.shields.io/badge/ca_central_1-latest-blue?logo=amazonaws)](https://ca-central-1.console.aws.amazon.com/cloudformation/home?region=ca-central-1#/stacks/create/review?templateURL=https://observeinc-ca-central-1.s3.amazonaws.com/apps/latest/forwarder/packaged.yaml) |
| [![Static Badge](https://img.shields.io/badge/sa_east_1-latest-blue?logo=amazonaws)](https://sa-east-1.console.aws.amazon.com/cloudformation/home?region=sa-east-1#/stacks/create/review?templateURL=https://observeinc-sa-east-1.s3.amazonaws.com/apps/latest/stack.yaml) | [![Static Badge](https://img.shields.io/badge/sa_east_1-latest-blue?logo=amazonaws)](https://sa-east-1.console.aws.amazon.com/cloudformation/home?region=sa-east-1#/stacks/create/review?templateURL=https://observeinc-sa-east-1.s3.amazonaws.com/apps/latest/config/packaged.yaml) | [![Static Badge](https://img.shields.io/badge/sa_east_1-latest-blue?logo=amazonaws)](https://sa-east-1.console.aws.amazon.com/cloudformation/home?region=sa-east-1#/stacks/create/review?templateURL=https://observeinc-sa-east-1.s3.amazonaws.com/apps/latest/forwarder/packaged.yaml) |
| [![Static Badge](https://img.shields.io/badge/ap_southeast_1-latest-blue?logo=amazonaws)](https://ap-southeast-1.console.aws.amazon.com/cloudformation/home?region=ap-southeast-1#/stacks/create/review?templateURL=https://observeinc-ap-southeast-1.s3.amazonaws.com/apps/latest/stack.yaml) | [![Static Badge](https://img.shields.io/badge/ap_southeast_1-latest-blue?logo=amazonaws)](https://ap-southeast-1.console.aws.amazon.com/cloudformation/home?region=ap-southeast-1#/stacks/create/review?templateURL=https://observeinc-ap-southeast-1.s3.amazonaws.com/apps/latest/config/packaged.yaml) | [![Static Badge](https://img.shields.io/badge/ap_southeast_1-latest-blue?logo=amazonaws)](https://ap-southeast-1.console.aws.amazon.com/cloudformation/home?region=ap-southeast-1#/stacks/create/review?templateURL=https://observeinc-ap-southeast-1.s3.amazonaws.com/apps/latest/forwarder/packaged.yaml) |
| [![Static Badge](https://img.shields.io/badge/ap_southeast_2-latest-blue?logo=amazonaws)](https://ap-southeast-2.console.aws.amazon.com/cloudformation/home?region=ap-southeast-2#/stacks/create/review?templateURL=https://observeinc-ap-southeast-2.s3.amazonaws.com/apps/latest/stack.yaml) | [![Static Badge](https://img.shields.io/badge/ap_southeast_2-latest-blue?logo=amazonaws)](https://ap-southeast-2.console.aws.amazon.com/cloudformation/home?region=ap-southeast-2#/stacks/create/review?templateURL=https://observeinc-ap-southeast-2.s3.amazonaws.com/apps/latest/config/packaged.yaml) | [![Static Badge](https://img.shields.io/badge/ap_southeast_2-latest-blue?logo=amazonaws)](https://ap-southeast-2.console.aws.amazon.com/cloudformation/home?region=ap-southeast-2#/stacks/create/review?templateURL=https://observeinc-ap-southeast-2.s3.amazonaws.com/apps/latest/forwarder/packaged.yaml) |
| [![Static Badge](https://img.shields.io/badge/eu_central_1-latest-blue?logo=amazonaws)](https://eu-central-1.console.aws.amazon.com/cloudformation/home?region=eu-central-1#/stacks/create/review?templateURL=https://observeinc-eu-central-1.s3.amazonaws.com/apps/latest/stack.yaml) | [![Static Badge](https://img.shields.io/badge/eu_central_1-latest-blue?logo=amazonaws)](https://eu-central-1.console.aws.amazon.com/cloudformation/home?region=eu-central-1#/stacks/create/review?templateURL=https://observeinc-eu-central-1.s3.amazonaws.com/apps/latest/config/packaged.yaml) | [![Static Badge](https://img.shields.io/badge/eu_central_1-latest-blue?logo=amazonaws)](https://eu-central-1.console.aws.amazon.com/cloudformation/home?region=eu-central-1#/stacks/create/review?templateURL=https://observeinc-eu-central-1.s3.amazonaws.com/apps/latest/forwarder/packaged.yaml) |
| [![Static Badge](https://img.shields.io/badge/us_east_1-latest-blue?logo=amazonaws)](https://us-east-1.console.aws.amazon.com/cloudformation/home?region=us-east-1#/stacks/create/review?templateURL=https://observeinc-us-east-1.s3.amazonaws.com/apps/latest/stack.yaml) | [![Static Badge](https://img.shields.io/badge/us_east_1-latest-blue?logo=amazonaws)](https://us-east-1.console.aws.amazon.com/cloudformation/home?region=us-east-1#/stacks/create/review?templateURL=https://observeinc-us-east-1.s3.amazonaws.com/apps/latest/config/packaged.yaml) | [![Static Badge](https://img.shields.io/badge/us_east_1-latest-blue?logo=amazonaws)](https://us-east-1.console.aws.amazon.com/cloudformation/home?region=us-east-1#/stacks/create/review?templateURL=https://observeinc-us-east-1.s3.amazonaws.com/apps/latest/forwarder/packaged.yaml) |
| [![Static Badge](https://img.shields.io/badge/us_east_2-latest-blue?logo=amazonaws)](https://us-east-2.console.aws.amazon.com/cloudformation/home?region=us-east-2#/stacks/create/review?templateURL=https://observeinc-us-east-2.s3.amazonaws.com/apps/latest/stack.yaml) | [![Static Badge](https://img.shields.io/badge/us_east_2-latest-blue?logo=amazonaws)](https://us-east-2.console.aws.amazon.com/cloudformation/home?region=us-east-2#/stacks/create/review?templateURL=https://observeinc-us-east-2.s3.amazonaws.com/apps/latest/config/packaged.yaml) | [![Static Badge](https://img.shields.io/badge/us_east_2-latest-blue?logo=amazonaws)](https://us-east-2.console.aws.amazon.com/cloudformation/home?region=us-east-2#/stacks/create/review?templateURL=https://observeinc-us-east-2.s3.amazonaws.com/apps/latest/forwarder/packaged.yaml) |
| [![Static Badge](https://img.shields.io/badge/us_west_1-latest-blue?logo=amazonaws)](https://us-west-1.console.aws.amazon.com/cloudformation/home?region=us-west-1#/stacks/create/review?templateURL=https://observeinc-us-west-1.s3.amazonaws.com/apps/latest/stack.yaml) | [![Static Badge](https://img.shields.io/badge/us_west_1-latest-blue?logo=amazonaws)](https://us-west-1.console.aws.amazon.com/cloudformation/home?region=us-west-1#/stacks/create/review?templateURL=https://observeinc-us-west-1.s3.amazonaws.com/apps/latest/config/packaged.yaml) | [![Static Badge](https://img.shields.io/badge/us_west_1-latest-blue?logo=amazonaws)](https://us-west-1.console.aws.amazon.com/cloudformation/home?region=us-west-1#/stacks/create/review?templateURL=https://observeinc-us-west-1.s3.amazonaws.com/apps/latest/forwarder/packaged.yaml) |
| [![Static Badge](https://img.shields.io/badge/us_west_2-latest-blue?logo=amazonaws)](https://us-west-2.console.aws.amazon.com/cloudformation/home?region=us-west-2#/stacks/create/review?templateURL=https://observeinc-us-west-2.s3.amazonaws.com/apps/latest/stack.yaml) | [![Static Badge](https://img.shields.io/badge/us_west_2-latest-blue?logo=amazonaws)](https://us-west-2.console.aws.amazon.com/cloudformation/home?region=us-west-2#/stacks/create/review?templateURL=https://observeinc-us-west-2.s3.amazonaws.com/apps/latest/config/packaged.yaml) | [![Static Badge](https://img.shields.io/badge/us_west_2-latest-blue?logo=amazonaws)](https://us-west-2.console.aws.amazon.com/cloudformation/home?region=us-west-2#/stacks/create/review?templateURL=https://observeinc-us-west-2.s3.amazonaws.com/apps/latest/forwarder/packaged.yaml) |

## Getting Started

To begin using these applications, you'll need to have the AWS CLI and SAM CLI installed and configured. See below for quick instructions on building and deploying an application. For a full development guide, check out the `DEVELOPER.md` file.

### Prerequisites

- AWS CLI
- SAM CLI
- Docker (optional, for linting and local testing)

### Building and Deploying an Application

Navigate to an application's directory under `apps/` and use the SAM CLI to build and deploy:

```sh
cd apps/forwarder
sam build
sam deploy --guided
```

For more detailed instructions on building, deploying, and publishing applications, please see the corresponding documentation in the `docs` folder.

## Testing

To run tests, use the Go tooling:

```sh
go test ./...
```

For more comprehensive testing instructions, please refer to `DEVELOPER.md`.

## Documentation

Each SAM application has its own documentation, providing specific details and usage instructions:

- [Stack](docs/stack.md)
- [Config](docs/config.md)
- [Firehose](docs/firehose.md)
- [Forwarder](docs/forwarder.md)
- [LogWriter](docs/logwriter.md)

For development practices, build and release processes, and testing workflows, see the `DEVELOPER.md` file.

## Contributing

We welcome contributions from the community. For more information on the contribution process, commit message standards, and branch naming conventions, please see our [CONTRIBUTING.md](CONTRIBUTING.md). For information on how to develop, please read through the [DEVELOPER.md](DEVELOPER.md) file and the documentation for the specific application you are interested in.
