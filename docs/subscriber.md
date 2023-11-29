# Observe Subscriber Application

The Observe Subscriber application is an AWS SAM application that subscribes CloudWatch Log Groups to a supported destination ARN, such as Kinesis Firehose or Lambda. It operates with two types of requests: subscription requests and discovery requests.

## Configuration

The subscriber Lambda function manages subscription filters for log groups and uses the following environment variables for configuration:

| Environment Variable      | Description |
|---------------------------|-------------|
| `FILTER_NAME`             | (Required) Name for the subscription filter. Any existing filters with this prefix will be removed. |
| `FILTER_PATTERN`          | Pattern for the subscription filter. See [AWS documentation](https://docs.aws.amazon.com/AmazonCloudWatch/latest/logs/SubscriptionFilters.html) for details. |
| `DESTINATION_ARN`         | Destination ARN for the subscription filter. If empty, any filters with `FILTER_NAME` will be removed. |
| `ROLE_ARN`                | Role ARN, required if `DESTINATION_ARN` is set. |

The scope of log groups the Lambda function applies to is determined by:

| Environment Variable      | Description |
|---------------------------|-------------|
| `LOG_GROUP_NAME_PATTERNS` | Comma-separated list of patterns to match log group names for subscription. Case-sensitive substring search is used. |
| `LOG_GROUP_NAME_PREFIXES` | Comma-separated list of prefixes to match log group names for subscription. |

**Note**: If neither `LOG_GROUP_NAME_PATTERNS` nor `LOG_GROUP_NAME_PREFIXES` are provided, the Lambda function will not operate on any log groups. It requires explicit patterns or prefixes to define its scope of operation.

## Subscription Request

To explicitly subscribe a set of log groups, invoke the Lambda function with a subscription request like the following:

```json
{
    "subscribe": {
        "logGroups": [
            {"logGroupName": "/aws/foo/example"},
            {"logGroupName": "/aws/bar/example"}
        ]
    }
}
```

### Response Format

The Lambda function returns statistics related to the processing of the log groups:

```json
{
    "subscription": {
        "deleted": 0,
        "updated": 0,
        "skipped": 0,
        "processed": 2
    }
}
```

Counters reflect the processing outcome for the log groups.

## Discovery Request

To subscribe log groups matching specific patterns or prefixes, send a discovery request. For example:

```json
{
    "discover": {
        "logGroupNamePatterns": ["prod"],
        "logGroupNamePrefixes": ["/aws/lambda"]
    }
}
```

This will list all log groups containing "prod" or prefixed with "/aws/lambda". To subscribe to all log groups, use the wildcard "*":

```json
{
    "discover": {
        "logGroupNamePatterns": ["*"]
    }
}
```

### Response Format

The function responds with statistics related to the listed log groups:

```json
{
    "discovery": {
        "logGroupCount": 3,
        "requestCount": 2
    }
}
```

### Inlining Subscriptions

To perform subscriptions in the same invocation as a discovery request, include the `inline` option:

```json
{
    "discover": {
        "inline": true
    }
}
```

The successful invocation response will include subscription stats embedded within the discovery stats.

---

## Additional Notes

- **Inline Subscriptions**: The `inline` option can be useful for immediate subscription after discovery but may increase the invocation duration.
- **SQS Queue Usage**: By default, if an SQS queue is provided, the Lambda function will fan out subscription requests for better scalability and management.
- **IAM Role**: The role specified in `ROLE_ARN` should have the necessary permissions to manage CloudWatch Logs and the destination resource.
- **Deployment and Updates**: For deployment instructions, refer to the main `README.md` and `DEVELOPER.md` documents. When updating the application, remember to adjust the `SemanticVersion` in `template.yaml` to reflect the changes.

Please refer to the provided `template.yaml` for the complete definition of the SAM application and to customize the deployment to fit your requirements.
