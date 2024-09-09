// Code generated by smithy-go-codegen DO NOT EDIT.

package cloudwatchlogs

import (
	"context"
	"fmt"
	awsmiddleware "github.com/aws/aws-sdk-go-v2/aws/middleware"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
	"github.com/aws/smithy-go/middleware"
	smithyhttp "github.com/aws/smithy-go/transport/http"
)

// Creates or updates a metric filter and associates it with the specified log
// group. With metric filters, you can configure rules to extract metric data from
// log events ingested through [PutLogEvents].
//
// The maximum number of metric filters that can be associated with a log group is
// 100.
//
// Using regular expressions to create metric filters is supported. For these
// filters, there is a quotas of quota of two regular expression patterns within a
// single filter pattern. There is also a quota of five regular expression patterns
// per log group. For more information about using regular expressions in metric
// filters, see [Filter pattern syntax for metric filters, subscription filters, filter log events, and Live Tail].
//
// When you create a metric filter, you can also optionally assign a unit and
// dimensions to the metric that is created.
//
// Metrics extracted from log events are charged as custom metrics. To prevent
// unexpected high charges, do not specify high-cardinality fields such as
// IPAddress or requestID as dimensions. Each different value found for a
// dimension is treated as a separate metric and accrues charges as a separate
// custom metric.
//
// CloudWatch Logs might disable a metric filter if it generates 1,000 different
// name/value pairs for your specified dimensions within one hour.
//
// You can also set up a billing alarm to alert you if your charges are higher
// than expected. For more information, see [Creating a Billing Alarm to Monitor Your Estimated Amazon Web Services Charges].
//
// [Creating a Billing Alarm to Monitor Your Estimated Amazon Web Services Charges]: https://docs.aws.amazon.com/AmazonCloudWatch/latest/monitoring/monitor_estimated_charges_with_cloudwatch.html
// [PutLogEvents]: https://docs.aws.amazon.com/AmazonCloudWatchLogs/latest/APIReference/API_PutLogEvents.html
// [Filter pattern syntax for metric filters, subscription filters, filter log events, and Live Tail]: https://docs.aws.amazon.com/AmazonCloudWatch/latest/logs/FilterAndPatternSyntax.html
func (c *Client) PutMetricFilter(ctx context.Context, params *PutMetricFilterInput, optFns ...func(*Options)) (*PutMetricFilterOutput, error) {
	if params == nil {
		params = &PutMetricFilterInput{}
	}

	result, metadata, err := c.invokeOperation(ctx, "PutMetricFilter", params, optFns, c.addOperationPutMetricFilterMiddlewares)
	if err != nil {
		return nil, err
	}

	out := result.(*PutMetricFilterOutput)
	out.ResultMetadata = metadata
	return out, nil
}

type PutMetricFilterInput struct {

	// A name for the metric filter.
	//
	// This member is required.
	FilterName *string

	// A filter pattern for extracting metric data out of ingested log events.
	//
	// This member is required.
	FilterPattern *string

	// The name of the log group.
	//
	// This member is required.
	LogGroupName *string

	// A collection of information that defines how metric data gets emitted.
	//
	// This member is required.
	MetricTransformations []types.MetricTransformation

	noSmithyDocumentSerde
}

type PutMetricFilterOutput struct {
	// Metadata pertaining to the operation's result.
	ResultMetadata middleware.Metadata

	noSmithyDocumentSerde
}

func (c *Client) addOperationPutMetricFilterMiddlewares(stack *middleware.Stack, options Options) (err error) {
	if err := stack.Serialize.Add(&setOperationInputMiddleware{}, middleware.After); err != nil {
		return err
	}
	err = stack.Serialize.Add(&awsAwsjson11_serializeOpPutMetricFilter{}, middleware.After)
	if err != nil {
		return err
	}
	err = stack.Deserialize.Add(&awsAwsjson11_deserializeOpPutMetricFilter{}, middleware.After)
	if err != nil {
		return err
	}
	if err := addProtocolFinalizerMiddlewares(stack, options, "PutMetricFilter"); err != nil {
		return fmt.Errorf("add protocol finalizers: %v", err)
	}

	if err = addlegacyEndpointContextSetter(stack, options); err != nil {
		return err
	}
	if err = addSetLoggerMiddleware(stack, options); err != nil {
		return err
	}
	if err = addClientRequestID(stack); err != nil {
		return err
	}
	if err = addComputeContentLength(stack); err != nil {
		return err
	}
	if err = addResolveEndpointMiddleware(stack, options); err != nil {
		return err
	}
	if err = addComputePayloadSHA256(stack); err != nil {
		return err
	}
	if err = addRetry(stack, options); err != nil {
		return err
	}
	if err = addRawResponseToMetadata(stack); err != nil {
		return err
	}
	if err = addRecordResponseTiming(stack); err != nil {
		return err
	}
	if err = addClientUserAgent(stack, options); err != nil {
		return err
	}
	if err = smithyhttp.AddErrorCloseResponseBodyMiddleware(stack); err != nil {
		return err
	}
	if err = smithyhttp.AddCloseResponseBodyMiddleware(stack); err != nil {
		return err
	}
	if err = addSetLegacyContextSigningOptionsMiddleware(stack); err != nil {
		return err
	}
	if err = addTimeOffsetBuild(stack, c); err != nil {
		return err
	}
	if err = addUserAgentRetryMode(stack, options); err != nil {
		return err
	}
	if err = addOpPutMetricFilterValidationMiddleware(stack); err != nil {
		return err
	}
	if err = stack.Initialize.Add(newServiceMetadataMiddleware_opPutMetricFilter(options.Region), middleware.Before); err != nil {
		return err
	}
	if err = addRecursionDetection(stack); err != nil {
		return err
	}
	if err = addRequestIDRetrieverMiddleware(stack); err != nil {
		return err
	}
	if err = addResponseErrorMiddleware(stack); err != nil {
		return err
	}
	if err = addRequestResponseLogging(stack, options); err != nil {
		return err
	}
	if err = addDisableHTTPSMiddleware(stack, options); err != nil {
		return err
	}
	return nil
}

func newServiceMetadataMiddleware_opPutMetricFilter(region string) *awsmiddleware.RegisterServiceMetadata {
	return &awsmiddleware.RegisterServiceMetadata{
		Region:        region,
		ServiceID:     ServiceID,
		OperationName: "PutMetricFilter",
	}
}
