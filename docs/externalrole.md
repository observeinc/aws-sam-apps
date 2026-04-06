# Observe External Role

The External Role template sets up an IAM role that can be assumed by Observe in order to collect metrics and API data (for example, CloudWatch metrics polling). Optionally, it can deploy a **PollerConfigurator** Lambda and custom resource when poller automation parameters are provided.

The template is **plain CloudFormation** (no SAM transform). Lambda functions load code from S3 using `LambdaS3BucketPrefix` and `LambdaS3Key`; released `externalrole.yaml` artifacts embed sensible defaults for those parameters.

## Template Configuration

### Parameters

#### Role (required for the assumable IAM role)

| Parameter | Type | Description |
|-----------|------|-------------|
| **`ObserveAwsAccountId`** | String | Observe AWS account ID that is allowed to assume this role. |
| **`AllowedActions`** | CommaDelimitedList | IAM actions that the Observe account may execute when assuming the role. |
| **`DatastreamIds`** | CommaDelimitedList | Datastream IDs where data will be ingested. Restricts use of the external ID on the role. |
| `NameOverride` | String | IAM role name override; if empty, the stack name is used. |
| **`PrimaryRegion`** | String | Region in which this stack **creates** the global IAM role. Must equal **`AWS::Region`** for the stack instance where the role should exist. Default `us-west-2` is only correct when deploying standalone in `us-west-2`; otherwise set this to your deploy region. When nested under the [Observe collection stack](stack.md), the parent sets this to the stack region automatically. |

Only stack instances where **`AWS::Region`** equals **`PrimaryRegion`** create the IAM role and emit **`RoleArn`**; other regions skip role creation (for example multi-region StackSets).

#### Poller automation (optional)

These parameters enable the PollerConfigurator Lambda and custom resource when **`PollerConfigURI`** is non-empty **and** Lambda code parameters are set (released templates embed defaults).

Enabling PollerConfigurator has the net effect of **creating a poller in the Observe backend for you** (the Lambda and custom resource register the poller against Observe using the GraphQL API). If PollerConfigurator is **not** enabled, this template does **not** create a poller—you must **create the poller separately** in Observe and point it at the IAM role. That latter flow is what you get when you deploy the **collection stack** from the **AWS integration** in the Observe app ([`stack.yaml`](stack.md)): the nested external-role stack only provisions the assumable IAM role; the poller is set up in the product independently of CloudFormation.

| Parameter | Type | Description |
|-----------|------|-------------|
| `ObserveCustomerAccountId` | String | Observe customer account ID for the GraphQL API. |
| `ObserveDomainName` | String | Observe domain name for the GraphQL API. |
| `WorkspaceID` | String | Observe workspace ID where the poller is configured. |
| `GQLToken` | String | Token with poller CRUD permissions (stored in Secrets Manager when the configurator is deployed). |
| `PollerConfigURI` | String | S3 URI of the poller configuration JSON. |
| `UpdateTimestamp` | String | Bump to force the PollerConfigurator custom resource to re-run. |

#### Lambda code (packaged deployments)

| Parameter | Type | Description |
|-----------|------|-------------|
| `LambdaS3BucketPrefix` | String | Prefix for the S3 bucket that holds Lambda ZIPs; bucket is `{prefix}-{region}` (for example `observeinc-us-west-2`). |
| `LambdaS3Key` | String | S3 key for the PollerConfigurator Lambda ZIP. |

### Outputs

| Output | Description |
|--------|-------------|
| `RoleArn` | IAM role ARN Observe assumes to pull data. Only not emitted when the **`PrimaryRegion`** of the StackSet is different from the region the stack is being deployed in. |
