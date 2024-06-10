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

// Creates or updates a logical delivery destination. A delivery destination is an
// Amazon Web Services resource that represents an Amazon Web Services service that
// logs can be sent to. CloudWatch Logs, Amazon S3, and Firehose are supported as
// logs delivery destinations.
//
// To configure logs delivery between a supported Amazon Web Services service and
// a destination, you must do the following:
//
//   - Create a delivery source, which is a logical object that represents the
//     resource that is actually sending the logs. For more information, see [PutDeliverySource].
//
//   - Use PutDeliveryDestination to create a delivery destination, which is a
//     logical object that represents the actual delivery destination.
//
//   - If you are delivering logs cross-account, you must use [PutDeliveryDestinationPolicy]in the destination
//     account to assign an IAM policy to the destination. This policy allows delivery
//     to that destination.
//
//   - Use CreateDelivery to create a delivery by pairing exactly one delivery
//     source and one delivery destination. For more information, see [CreateDelivery].
//
// You can configure a single delivery source to send logs to multiple
// destinations by creating multiple deliveries. You can also create multiple
// deliveries to configure multiple delivery sources to send logs to the same
// delivery destination.
//
// Only some Amazon Web Services services support being configured as a delivery
// source. These services are listed as Supported [V2 Permissions] in the table at [Enabling logging from Amazon Web Services services.]
//
// If you use this operation to update an existing delivery destination, all the
// current delivery destination parameters are overwritten with the new parameter
// values that you specify.
//
// [PutDeliverySource]: https://docs.aws.amazon.com/AmazonCloudWatchLogs/latest/APIReference/API_PutDeliverySource.html
// [Enabling logging from Amazon Web Services services.]: https://docs.aws.amazon.com/AmazonCloudWatch/latest/logs/AWS-logs-and-resource-policy.html
// [CreateDelivery]: https://docs.aws.amazon.com/AmazonCloudWatchLogs/latest/APIReference/API_CreateDelivery.html
// [PutDeliveryDestinationPolicy]: https://docs.aws.amazon.com/AmazonCloudWatchLogs/latest/APIReference/API_PutDeliveryDestinationPolicy.html
func (c *Client) PutDeliveryDestination(ctx context.Context, params *PutDeliveryDestinationInput, optFns ...func(*Options)) (*PutDeliveryDestinationOutput, error) {
	if params == nil {
		params = &PutDeliveryDestinationInput{}
	}

	result, metadata, err := c.invokeOperation(ctx, "PutDeliveryDestination", params, optFns, c.addOperationPutDeliveryDestinationMiddlewares)
	if err != nil {
		return nil, err
	}

	out := result.(*PutDeliveryDestinationOutput)
	out.ResultMetadata = metadata
	return out, nil
}

type PutDeliveryDestinationInput struct {

	// A structure that contains the ARN of the Amazon Web Services resource that will
	// receive the logs.
	//
	// This member is required.
	DeliveryDestinationConfiguration *types.DeliveryDestinationConfiguration

	// A name for this delivery destination. This name must be unique for all delivery
	// destinations in your account.
	//
	// This member is required.
	Name *string

	// The format for the logs that this delivery destination will receive.
	OutputFormat types.OutputFormat

	// An optional list of key-value pairs to associate with the resource.
	//
	// For more information about tagging, see [Tagging Amazon Web Services resources]
	//
	// [Tagging Amazon Web Services resources]: https://docs.aws.amazon.com/general/latest/gr/aws_tagging.html
	Tags map[string]string

	noSmithyDocumentSerde
}

type PutDeliveryDestinationOutput struct {

	// A structure containing information about the delivery destination that you just
	// created or updated.
	DeliveryDestination *types.DeliveryDestination

	// Metadata pertaining to the operation's result.
	ResultMetadata middleware.Metadata

	noSmithyDocumentSerde
}

func (c *Client) addOperationPutDeliveryDestinationMiddlewares(stack *middleware.Stack, options Options) (err error) {
	if err := stack.Serialize.Add(&setOperationInputMiddleware{}, middleware.After); err != nil {
		return err
	}
	err = stack.Serialize.Add(&awsAwsjson11_serializeOpPutDeliveryDestination{}, middleware.After)
	if err != nil {
		return err
	}
	err = stack.Deserialize.Add(&awsAwsjson11_deserializeOpPutDeliveryDestination{}, middleware.After)
	if err != nil {
		return err
	}
	if err := addProtocolFinalizerMiddlewares(stack, options, "PutDeliveryDestination"); err != nil {
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
	if err = addOpPutDeliveryDestinationValidationMiddleware(stack); err != nil {
		return err
	}
	if err = stack.Initialize.Add(newServiceMetadataMiddleware_opPutDeliveryDestination(options.Region), middleware.Before); err != nil {
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

func newServiceMetadataMiddleware_opPutDeliveryDestination(region string) *awsmiddleware.RegisterServiceMetadata {
	return &awsmiddleware.RegisterServiceMetadata{
		Region:        region,
		ServiceID:     ServiceID,
		OperationName: "PutDeliveryDestination",
	}
}
