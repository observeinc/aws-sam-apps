# Observe LogWriter Application

The Observe LogWriter application is an AWS SAM application that writes CloudWatch Log Groups to an S3 bucket. 

Additionally, the stack is capable of subscribing log groups and provides a method for automatically triggering subscription through Eventbridge rules.

## Configuration

The subscriber Lambda function manages subscription filters for log groups and uses the following environment variables for configuration:

| Environment Variable      | Description |
|---------------------------|-------------|
| `FILTER_NAME`             | (Required) Name for the subscription filter. Any existing filters with this prefix will be removed. |
| `FILTER_PATTERN`          | Pattern for the subscription filter. See [AWS documentation](https://docs.aws.amazon.com/AmazonCloudWatch/latest/logs/SubscriptionFilters.html) for details. |

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

```json
{
    "discovery": {
        "logGroupCount": 3,
        "requestCount": 2,
        "subscription": {
            "deleted": 0,
            "updated": 0,
            "skipped": 0,
            "processed": 3
        }
    }
}
```

## Automatic subscription through Eventbridge rules

The stack optionally installs eventbridge rules which automatically subscribe log groups the the configured destination. To enable this feature, you must set the `DiscoveryRate` parameter to a valid [AWS EventBridge rate expression](https://docs.aws.amazon.com/eventbridge/latest/userguide/eb-rate-expressions.html) (e.g. `1 hour`).

If this parameter is set, two EventBridge rules are installed:

- a discovery request that will be fire at the desired rate,
- a subscription request will be fired on log group creation. This rule will only fire if CloudTrail is configured within the account and region our subscriber is running in.

Both rules will send requests to the SQS queue, which in turn are consumed by the subscriber lambda.
