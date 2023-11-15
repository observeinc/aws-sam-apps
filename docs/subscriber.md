# Observe Subscriber

The subscriber stack subscribes CloudWatch Log Groups to a supported destination ARN (either Kinesis Firehose or Lambda). It supports two request types:

- subscription requests contain a list of log groups which we wish to subscribe to our destination.
- discovery requests contain a list of filters which are used to generate subscription requests.

## Configuration

The subscriber lambda is responsible for managing subscription filters for a set of log groups.
The subscription filter will be configured according the following environment variables:

| Environment Variable | Description                                                                                                                                   |
|----------------------|-----------------------------------------------------------------------------------------------------------------------------------------------|
| `FILTER_NAME`        | **Required**. Subscription filter name. Existing filters that have this name as a prefix will be removed.                                     |
| `FILTER_PATTERN`     | Subscription filter pattern. Refer to [AWS documentation](https://docs.aws.amazon.com/AmazonCloudWatch/latest/logs/SubscriptionFilters.html). |
| `DESTINATION_ARN`    | Destination ARN. If empty, any matching subscription filter named `FILTER_NAME` will be removed.                                              |
| `ROLE_ARN`           | Role ARN. Can only be set if `DESTINATION_ARN` is also set                                                                                    |

Additionally, the set of log groups the lambda is applicable to is controlled through the following variables:

| Environment Variable      | Description                                                                                                                                                                                          |
|---------------------------|------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `LOG_GROUP_NAME_PATTERNS` | Comma separated list of patterns. If not empty, the lambda function will only apply to log groups that have names that match one of the provided strings based on a case-sensitive substring search. |
| `LOG_GROUP_NAME_PREFIXES` | Comma separated list of prefixes. If not empty, the lambda function will only apply to log groups that start with a provided string.                                                                 |

If neither `LOG_GROUP_NAME_PATTERNS` or `LOG_GROUP_NAME_PREFIXES` is provided, the subscriber will operate across all log groups.


## Subscription request

You can subscribe a set of log groups by invoking the lambda function via a subscription request, e.g:

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
