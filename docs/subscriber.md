# Observe Subscriber

The subscriber stack subscribes CloudWatch Log Groups to a supported destination ARN (either Kinesis Firehose or Lambda). It supports two request types:

- subscription requests contain a list of log groups which we wish to subscribe to our destination.
- discovery requests contain a list of filters which are used to generate subscription requests.

Additionally, the stack provides a method for automatically triggering subscription through Eventbridge rules.

## Configuration

The subscriber lambda is responsible for managing subscription filters for a set of log groups.
The subscription filter will be configured according the following environment variables:

| Environment Variable | Description                                                                                                                                   |
|----------------------|-----------------------------------------------------------------------------------------------------------------------------------------------|
| `FILTER_NAME`        | **Required**. Subscription filter name. Existing filters that have this name as a prefix will be removed.                                     |
| `FILTER_PATTERN`     | Subscription filter pattern. Refer to [AWS documentation](https://docs.aws.amazon.com/AmazonCloudWatch/latest/logs/SubscriptionFilters.html). |
| `DESTINATION_ARN`    | Destination ARN. If empty, any matching subscription filter named `FILTER_NAME` will be removed.                                              |
| `ROLE_ARN`           | Role ARN. Can only be set if `DESTINATION_ARN` is also set.                                                                                   |

Additionally, the set of log groups the lambda is applicable to is controlled through the following variables:

| Environment Variable      | Description                                                                                                                                                                                          |
|---------------------------|------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `LOG_GROUP_NAME_PATTERNS` | Comma separated list of patterns. If not empty, the lambda function will only apply to log groups that have names that match one of the provided strings based on a case-sensitive substring search. |
| `LOG_GROUP_NAME_PREFIXES` | Comma separated list of prefixes. If not empty, the lambda function will only apply to log groups that start with a provided string.                                                                 |

If neither `LOG_GROUP_NAME_PATTERNS` or `LOG_GROUP_NAME_PREFIXES` is provided, the subscriber will operate across all log groups.


## Subscription request

You can subscribe an explicit set of log groups by invoking the lambda function via a subscription request, e.g:

```
{
    "subscribe": {
        "logGroups": [
            {
                "logGroupName": "/aws/foo/example"
            },
            {
                "logGroupName": "/aws/bar/example"
            }
        ]
    }
}
```

### Response format

The function will respond with stats associated to the processing of the log groups:

```
{
    "subscription":	{
        "deleted": 0,
        "updated": 0,
        "skipped": 0,
        "processed": 2
    }
}
```

The counters reflect how the log groups were processed:  

| Counter     | Description                                                                                                                                                                                             |
|-------------|---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `processed` | The total number of log groups processed.                                                                                                                                                               |
| `skipped`   | The number of log groups that were ignored. Either the log group no longer exists, or the log group name does not match the provided filters in `LOG_GROUP_NAME_PATTERNS` or `LOG_GROUP_NAME_PREFIXES`. |
| `updated`   | The number of subscription filters which were updated. This maps to the total number of calls to the `logs:PutSubscriptionFilter` endpoint.                                                             |
| `deleted`   | The number of subscription filters which were deleted. This maps to the total number of calls to the `logs:DeleteSubscriptionFilter` endpoint.                                                          |

## Discovery request

You can subscribe to log groups matching a set of patterns or prefixes by sending a discovery request. The following request will ask the lambda function to list all log groups containing the term `prod` or prefixed by the term `/aws/lambda`:

```json
{
    "discover": {
        "logGroupNamePatterns": [ "prod" ],
        "logGroupNamePrefixes": [ "/aws/lambda" ]
    }
}
```

The lambda function will issue a `logs:DescribeLogGroups` request for each provided pattern or prefix. The equivalent `awscli` commands for the above example request would be:

```
aws logs describe-log-groups --log-group-name-pattern prod
aws logs describe-log-groups --log-group-name-prefix /aws/lambda
```

To subscribe to all log groups, a wildcard can be provided to either `logGroupNamePatterns` or `logGroupNamePrefixes`. The following input: 

```json
{
    "discover": {
        "logGroupNamePatterns": [ "*" ]
    }
}
```

Will trigger a paginated request equivalent to the `awscli` command:

```shell
aws logs describe-log-groups
```


### Response format

The function will respond with stats associated to the listing of log groups:

```
{
    "discovery": {
        "logGroupCount": 3,
        "requestCount": 2,
    }
}
```

| Counter         | Description                                         |
|-----------------|-----------------------------------------------------|
| `logGroupCount` | The total number of log groups retrieved.       |
| `requestCount`  | The number of requests to the AWS API.          |


### Inlining subscriptions

By omission, if you provide an SQS queue the lambda function will use it to fan out subscription requests across multiple lambda invocations. If you instead wish to inline subscription to be performed in the same invocation as a discovery request, you can provide the `inline` option in your request: 

```json
{
    "discover": {
        "inline": true
    }
}
```

The response for a successful invocation will embed the corresponding subscription stats:

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
