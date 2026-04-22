# Troubleshooting Observe AWS Collection

## CLI Pitfalls

### Comma-separated values parsed as lists

```
Invalid type for parameter Parameters[1].ParameterValue, value: ['us-west-2', 'us-east-1'], type: <class 'list'>
```

**Cause**: The AWS CLI splits unquoted comma-separated values into JSON
arrays. For example, `ParameterValue=us-west-2,us-east-1` becomes a list.

**Fix**: Wrap the entire key=value pair in single quotes and the value in
double quotes:

```bash
# WRONG — CLI splits on the comma
--parameters ParameterKey=TargetRegions,ParameterValue=us-west-2,us-east-1

# RIGHT — value treated as a single string
--parameters 'ParameterKey=TargetRegions,ParameterValue="us-west-2,us-east-1"'
```

This applies to all `CommaDelimitedList` parameters: `LogGroupNamePatterns`,
`LogGroupNamePrefixes`, etc.

### Expired session tokens

```
An error occurred (ExpiredToken) when calling the DescribeStacks operation
```

**Cause**: AWS session credentials have expired (common with SSO/Britive
profiles).
**Fix**: Re-authenticate. This is not a stack error — just refresh credentials
and retry.

---

## Deployment Errors

### `LogGroupNamePatterns` validation failure

```
Each value of parameter 'LogGroupNamePatterns' must match pattern ^(\*|[a-zA-Z0-9-_\/]*)$
```

**Cause**: Customer used regex syntax (e.g. `.*`) in `LogGroupNamePatterns`.
This parameter does NOT accept arbitrary regex — only alphanumeric strings,
hyphens, underscores, forward slashes, and `*` wildcard.

**Fix**: Use `LogGroupNamePrefixes` instead for broad matching. For example,
`/aws` will match all log groups starting with `/aws`. Or use `*` to match
all log groups.

### Missing capabilities

```
Requires capabilities: [CAPABILITY_IAM, CAPABILITY_NAMED_IAM, CAPABILITY_AUTO_EXPAND]
```

**Fix**: Add all three to the `--capabilities` flag. `AUTO_EXPAND` is required
because the template uses the SAM transform and nested CloudFormation stacks.

### IAM role already exists

```
observe-collection already exists in stack arn:aws:cloudformation:...
```

**Cause**: `NameOverride` conflicts with an existing IAM role.
**Fix**: Choose a unique `NameOverride` value and update the filedrop in
Observe to match.

---

## Metrics Not Flowing

### Metric stream shows `Namespace: "Default"`

```bash
aws cloudwatch get-metric-stream --name "STACK-MetricStream-metric-stream-0" \
  --query 'IncludeFilters[].Namespace'
# Returns: ["Default"]
```

**Cause**: The `stack` template hardcodes `FilterUri` to `default.yaml`
(a no-op placeholder) when Lambda code is embedded (`DeployLambda` condition
is true). The MetricsConfigurator needs a `DatasourceID` to pull real filters.

**Fix**:
1. Create a Datasource in Observe with `awsCollectionStackConfig` containing
   desired metric namespaces
2. Update the stack with `DatasourceID`, `GQLToken`, `ObserveAccountID`,
   `ObserveDomainName`, and a new `UpdateTimestamp`
3. The `UpdateTimestamp` change is critical — without it, the custom resource
   won't re-invoke the MetricsConfigurator Lambda

### Metric stream is "running" but no data in bucket

**Diagnostic steps**:
1. Check the Firehose delivery stream destination:
   ```bash
   aws firehose describe-delivery-stream \
     --delivery-stream-name "STACK-MetricStream" \
     --query 'DeliveryStreamDescription.Destinations[0].S3DestinationDescription.{Bucket:BucketARN,Prefix:Prefix}'
   ```
2. Check the Firehose error logs:
   ```bash
   aws logs get-log-events \
     --log-group-name "/aws/firehose/STACK-MetricStream"
   ```
3. Check the MetricsConfigurator Lambda logs for errors:
   ```bash
   aws logs describe-log-groups \
     --log-group-name-prefix "/aws/lambda/STACK-MetricStream"
   ```
   Then read the latest log stream for that group.

**Common causes**:
- Firehose buffering interval is 60s — wait at least 2 minutes
- Metric stream filters include only invalid/empty namespaces
- Firehose IAM role lacks `s3:PutObject` on the collection bucket

### MetricsConfigurator Lambda not invoked on update

**Cause**: The `StackCreationUpdateCustomResource` only re-triggers when its
properties change. `UpdateTimestamp` is the mechanism to force this.

**Fix**: Always set `UpdateTimestamp` to a new value (e.g. `$(date +%s)`)
when updating DatasourceID, GQLToken, or any metrics configuration.

### MetricsConfigurator fails to reach Observe API

Check the Lambda logs for connection errors:
```
failed to retrieve datasource
```

**Common causes**:
- `ObserveDomainName` is wrong (e.g. `observe-o2.com` vs `observe-eng.com`)
- `ObserveAccountID` doesn't match the token's tenant
- `GQLToken` is expired or invalid
- Lambda doesn't have outbound internet access (VPC configuration)

---

## Logs Not Flowing

### No data in `cloudwatchlogs/` prefix

**Diagnostic steps**:
1. Check the LogWriter Firehose is active:
   ```bash
   aws firehose describe-delivery-stream \
     --delivery-stream-name "STACK-LogWriter" \
     --query 'DeliveryStreamDescription.DeliveryStreamStatus'
   ```
2. Check the Subscriber Lambda logs to verify log groups are being discovered:
   ```bash
   aws logs get-log-events \
     --log-group-name "/aws/lambda/STACK-LogWriter" \
     --log-stream-name "$(aws logs describe-log-streams \
       --log-group-name '/aws/lambda/STACK-LogWriter' \
       --order-by LastEventTime --descending --max-items 1 \
       --query 'logStreams[0].logStreamName' --output text)" \
     --limit 20 --query 'events[].message' --output text
   ```

**Common causes**:
- No log groups match the provided prefixes/patterns
- Subscription filter limit reached (AWS allows max 2 per log group)
- Subscriber Lambda doesn't have `logs:PutSubscriptionFilter` permission

### Forwarder Lambda errors

Check the Forwarder log group (stack output `ForwarderLogGroupName`):
```bash
aws logs get-log-events \
  --log-group-name "/aws/lambda/STACK_NAME" \
  --log-stream-name "LATEST_STREAM" \
  --limit 20 --query 'events[].message' --output text
```

**Common causes**:
- `DestinationUri` doesn't match filedrop configuration
- `NameOverride` role doesn't match what the filedrop expects
- The filedrop's IAM trust policy doesn't trust the Forwarder role ARN
  (check `ForwarderRoleArn` in stack outputs vs filedrop config)

---

## Updating the Stack

When updating parameters, use `UsePreviousValue=true` for unchanged params:

```bash
aws cloudformation update-stack \
  --stack-name observe-collection \
  --use-previous-template \
  --parameters \
    ParameterKey=DestinationUri,UsePreviousValue=true \
    ParameterKey=DataAccessPointArn,UsePreviousValue=true \
    ParameterKey=NameOverride,UsePreviousValue=true \
    ParameterKey=LogGroupNamePrefixes,UsePreviousValue=true \
    ParameterKey=DatasourceID,ParameterValue="NEW_DATASOURCE_ID" \
    ParameterKey=GQLToken,ParameterValue="NEW_TOKEN" \
    ParameterKey=ObserveAccountID,UsePreviousValue=true \
    ParameterKey=ObserveDomainName,UsePreviousValue=true \
    ParameterKey=UpdateTimestamp,ParameterValue="$(date +%s)" \
  --capabilities CAPABILITY_IAM CAPABILITY_NAMED_IAM CAPABILITY_AUTO_EXPAND \
  --region us-west-2
```

## Deleting the Stack

```bash
aws cloudformation delete-stack --stack-name observe-collection --region us-west-2
```

Some resources may have `DeletionPolicy: Retain` (e.g. log groups). Clean
these up manually if needed:
```bash
aws logs delete-log-group --log-group-name "/aws/lambda/STACK_NAME"
aws logs delete-log-group --log-group-name "/aws/lambda/STACK_NAME-LogWriter"
```
