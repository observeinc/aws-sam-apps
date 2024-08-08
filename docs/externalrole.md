# Observe External Role

The External Role template sets up an IAM role that can be assumed by Observe in order to collect metrics and API data.

## Template Configuration

### Parameters

The stack supports the following parameters:

| Parameter       | Type    | Description |
|-----------------|---------|-------------|
| **`ObserveAwsAccountId`** | String | Observe AWS Account ID which will be allowed to assume role. |
| **`AllowedActions`** | CommaDelimitedList | IAM actions that Observe account is allowed to execute. |
| **`DatastreamIds`** | CommaDelimitedList | Datastream IDs where data will be ingested to. This ensures Observe cannot assume this role outside of this context. |
| `NameOverride` | String | Name of IAM role expected by Poller. In the absence of a value, the stack name will be used. |

### Outputs

| Output       |  Description |
|-----------------|-------------|
| RoleArn | IAM Role ARN. This role will be assumed by Observe in order to pull data. |
