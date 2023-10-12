# SAM test

This repo attempts to validate:

- [x] Go code producing multiple binaries with unit tests
- [x] Multiple AWS SAM applications
- [x] An application that references other published applications


## Testing

The codebase looks like a standard golang project, so all the standard Go tooling should just work, e.g:

```
→ go test ./...
ok      github.com/observeinc/aws-sam-testing/cmd/ec2   (cached)
?       github.com/observeinc/aws-sam-testing/model/aws/ec2     [no test files]
?       github.com/observeinc/aws-sam-testing/model/aws/ec2/marshaller  [no test files]
ok      github.com/observeinc/aws-sam-testing/cmd/hello-world   (cached)
```

## Building and deploying

Each AWS SAM template lives under `apps/`. You can use the `sam` cli to build, invoke and deploy the cloudformation stack for testing.

```
→ cd apps/hello-world
→ sam build
Starting Build use cache
Valid cache found, copying previously built resources for following functions (HelloWorldFunction)

Build Succeeded

Built Artifacts  : .aws-sam/build
Built Template   : .aws-sam/build/template.yaml
```

## Publishing apps

Each SAM app can be packaged and published to the AWS Serverless Application Repository.

1. Bump the `SemanticVersion` in `template.yaml`:

```
Metadata:
  AWS::ServerlessRepo::Application:
    Name: hello-world-thing
    Description: A hello world
    Author: Observe Inc
    SpdxLicenseId: Apache-2.0
    ReadmeUrl: README.md
    HomePageUrl: https://github.com/observeinc/aws-sam-testing
    SemanticVersion: 0.0.2
    SourceCodeUrl: https://github.com/observeinc/aws-sam-testing
```

2. Package the template:

```
→ sam package --template-file template.yaml --output-template-file output.yaml

                Managed S3 bucket: aws-sam-cli-managed-default-samclisourcebucket-1d9p458evhgwf
                A different default S3 bucket can be set in samconfig.toml
                Or by specifying --s3-bucket explicitly.
File with same data already exists at 410049026555143131a24cfafb8eb862, skipping upload
        Uploading to bae5b6e5768f0a6e85e1b9c52dddcead  1021 / 1021  (100.00%)

Successfully packaged artifacts and wrote output template to file output.yaml.
Execute the following command to deploy the packaged template
sam deploy --template-file /Users/joao/Code/aws-sam-testing/apps/hello-world/output.yaml --stack-name <YOUR STACK NAME>
```

3. Publish the packaged template (`sam publish --template output.yaml`)

```
→ sam publish --template output.yaml

Publish Succeeded
The following metadata of application "arn:aws:serverlessrepo:us-west-2:739672403694:applications/hello-world-thing" has been updated:
{
  "Description": "A hello world",
  "Author": "Observe Inc",
  "ReadmeUrl": "s3://aws-sam-cli-managed-default-samclisourcebucket-1d9p458evhgwf/410049026555143131a24cfafb8eb862",
  "HomePageUrl": "https://github.com/observeinc/aws-sam-testing",
  "SemanticVersion": "0.0.2",
  "SourceCodeUrl": "https://github.com/observeinc/aws-sam-testing"
}
Click the link below to view your application in AWS console:
https://console.aws.amazon.com/serverlessrepo/home?region=us-west-2#/published-applications/arn:aws:serverlessrepo:us-west-2:739672403694:applications~hello-world-thing
```
## Release Workflow

Our release process is automated using GitHub Actions, ensuring consistency and reliability in each release.

### Release Steps

1. **Pre-release (Beta Releases on `main` branch):**
   Whenever changes are pushed to the `main` branch, our automated workflow triggers a beta release. This provides early access versions for testing and validation purposes.

2. **Full Release (Manual Trigger):**
   For creating an official release, manually trigger the release workflow from the GitHub Actions interface. This performs a full release.

3. **AWS SAM Build & Deployment:**
   - The AWS SAM application is built once at the beginning of the release phase to ensure consistency across regions.
   - AWS SAM resources are packaged and deployed across multiple AWS regions, specified in the `REGIONS` variable of our Makefile.

### Notes

- All semantic versioning is handled automatically by the `semantic-release` tool. This determines the version number based on the commit messages since the last release.
  
- Always ensure commit messages adhere to the [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/) standard, as our release tooling relies on this format.

- For region-specific AWS SAM builds or configurations, check the build artifacts located in `.aws-sam/build/<REGION_NAME>/`.
