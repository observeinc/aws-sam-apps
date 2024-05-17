# Observe LogWriter Application

The Observe LogWriter application is an AWS SAM application that writes CloudWatch Log Groups to an S3 bucket. 

Additionally, the stack is capable of subscribing log groups and provides a method for automatically triggering subscription through Eventbridge rules.

## Configuration parameters

| Parameter       | Type    | Description |
|-----------------|---------|-------------|
| **`BucketARN`** | String | S3 Bucket ARN to write log records to. |
| `Prefix` | String | Optional prefix to write log records to. |
| `LogGroupNamePatterns` | CommaDelimitedList | Comma separated list of patterns. We will only subscribe to log groups that have names matching one of the provided strings based on strings based on a case-sensitive substring search. To subscribe to all log groups, use the wildcard operator *. |
| `LogGroupNamePrefixes` | CommaDelimitedList | Comma separated list of prefixes. The lambda function will only apply to log groups that start with a provided string. To subscribe to all log groups, use the wildcard operator *. |
| `DiscoveryRate` | String | EventBridge rate expression for periodically triggering discovery. If not set, no eventbridge rules are configured. |
| `FilterName` | String | Subscription filter name. Existing filters that have this name as a prefix will be removed. |
| `FilterPattern` | String | Subscription filter pattern. |
| `NameOverride` | String | Name of Lambda function. |
| `BufferingInterval` | Number | Buffer incoming data for the specified period of time, in seconds, before delivering it to S3.  |
| `BufferingSize` | Number | Buffer incoming data to the specified size, in MiBs, before delivering it to S3.  |
| `NumWorkers` | String | Maximum number of concurrent workers when processing log groups. |
| `MemorySize` | String | The amount of memory, in megabytes, that your function has access to. |
| `Timeout` | String | The amount of time that Lambda allows a function to run before stopping it. The maximum allowed value is 900 seconds. |
| `DebugEndpoint` | String | Endpoint to send additional debug telemetry to. |
| `Verbosity` | String | Logging verbosity for Lambda. Highest log verbosity is 9. |

**Note**: If neither `LogGroupNamePatterns` nor `LogGroupNamePrefixes` are provided, the Lambda function will not operate on any log groups. It requires explicit patterns or prefixes to define its scope of operation.

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
