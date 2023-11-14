# Observe Subscriber

The subscriber stack subscribes CloudWatch Log Groups to a supported destination ARN (either Kinesis Firehose or Lambda). It supports two request types:

- subscription requests contain a list of log groups which we wish to subscribe to our destination.
- discovery requests contain a list of filters which are used to generate subscription requests.
