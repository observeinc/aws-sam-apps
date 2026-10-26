# X-Ray poller

The X-Ray poller reads traces from the X-Ray API and posts them to Observe as OTLP spans. It is a nested app of the [collection stack](stack.md). It stays off unless `EnableXRayPoller` is `true`.

API Gateway and other managed services write segments only to X-Ray. They do not send those segments to the X-Ray daemon, and there is no log subscription or metric stream for traces. The poller calls `GetTraceSummaries` and `BatchGetTraces` in the stack's region and posts OTLP/JSON to `https://<customer>.collect.<domain>/v2/otel/v1/traces`.

Traces do not go through the collection bucket or the Forwarder. Those paths land in a filedrop datastream and do not become spans.

## What you need first

- X-Ray tracing already enabled on the services you want. The stack does not turn tracing on, and it does not instrument the application.
- REST APIs (API Gateway v1) can emit X-Ray. HTTP APIs (API Gateway v2) cannot.
- A backend service joins the same trace only when it reads `X-Amzn-Trace-Id`. For an OpenTelemetry SDK that means `OTEL_PROPAGATORS=xray,tracecontext`.
- `ObserveAccountID`, `ObserveDomainName`, and `ObserveIngestToken` set on the collection stack. The token is a long-lived datastream ingest token. The nested stack writes it to Secrets Manager, and the Lambda reads that secret on each invocation. Revoke it from the Observe ingest-token page. The quick-create URL contains the token once, the same way it contains the GraphQL token, and this token does not expire after a day.

## Parameters

The parent stack passes these into the nested app. See [stack parameters](stack.md#parameters) for `EnableXRayPoller`, `XRayPollIntervalMinutes`, `XRayFilterExpression`, and `ObserveIngestToken`.

| Parameter | Description |
| --- | --- |
| `ObserveCustomerId` | Customer id used in the collect hostname. |
| `ObserveDomainName` | Observe domain, for example `observeinc.com`. |
| `ObserveIngestToken` | Ingest token secret. Required, and stored in Secrets Manager. |
| `PollIntervalMinutes` | `1`, `2`, or `5`. Default `1`. |
| `FilterExpression` | X-Ray filter. Default `service(id(type: "AWS::ApiGateway::Stage"))`. Empty reads every trace in the region. |
| `LambdaS3BucketPrefix` / `LambdaS3Key` | Location of the poller ZIP. Released templates fill these in. |
| `PermissionsBoundary` | Optional IAM permissions boundary. |

One poller runs per stack region. A StackSet that deploys the parent template deploys the poller in each region where the parent is deployed.

## How a poll works

Each invocation reads `(cursor, now - 90s]`. The 90 second lag is time for X-Ray to index a trace. The cursor is an SSM parameter, `/observe/<stack>/xray-poller/cursor`, and it advances only after a successful export (or when the window has no traces). The first poll looks back 60 seconds from the lagged end. A backlog longer than 15 minutes is drained 15 minutes at a time so one invocation cannot time out on days of traces.

`BatchGetTraces` follows `NextToken` and retries `UnprocessedTraceIds`. Reserved concurrency is 1. The function times out after 3 minutes and sends failed invocations to a dead-letter queue.

The role allows `xray:GetTraceSummaries` and `xray:BatchGetTraces` to read, `xray:PutTraceSegments` and `xray:PutTelemetryRecords` for the function's own active tracing, plus the queue, the cursor parameter, and the ingest-token secret. X-Ray does not support resource-level permissions, so the X-Ray statements use `*`. The other statements name a single resource.

The ingest-token secret is encrypted with a customer-managed KMS key that the nested stack creates, with yearly rotation turned on. The role can decrypt with that key only through Secrets Manager. The dead-letter queue uses SQS-managed server-side encryption.

The function traces itself in X-Ray. The poller adds `!service("<function name>")` to the filter so it does not forward its own traces.

## Limits

- Latency is about a minute, plus the 90 second indexing lag.
- The default filter is API Gateway. Widen `XRayFilterExpression` for other managed services. An empty filter also copies application traces that are not already in Observe.
- Inferred segments are dropped. Those are X-Ray's synthetic view of an OpenTelemetry span, and copying them would duplicate a span the SDK already exported.
- An API Gateway outgoing client subsegment can render as a gap in the waterfall. Stitching is by trace id: the backend span is a sibling that shares the trace, not a child that fills that gap.
- The X-Ray trace id `1-{time}-{rand}` becomes the 32-hex OTLP trace id, so a backend that extracted `X-Amzn-Trace-Id` joins the API Gateway span.
- `service.name` comes from the segment name, then its origin. API Gateway and a downstream service such as DynamoDB are separate services in the trace.
