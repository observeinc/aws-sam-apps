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

// Creates or updates a logical delivery source. A delivery source represents an
// Amazon Web Services resource that sends logs to an logs delivery destination.
// The destination can be CloudWatch Logs, Amazon S3, or Firehose.
//
// To configure logs delivery between a delivery destination and an Amazon Web
// Services service that is supported as a delivery source, you must do the
// following:
//
//   - Use PutDeliverySource to create a delivery source, which is a logical object
//     that represents the resource that is actually sending the logs.
//
//   - Use PutDeliveryDestination to create a delivery destination, which is a
//     logical object that represents the actual delivery destination. For more
//     information, see [PutDeliveryDestination].
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
// If you use this operation to update an existing delivery source, all the
// current delivery source parameters are overwritten with the new parameter values
// that you specify.
//
// [PutDeliveryDestination]: https://docs.aws.amazon.com/AmazonCloudWatchLogs/latest/APIReference/API_PutDeliveryDestination.html
// [Enabling logging from Amazon Web Services services.]: https://docs.aws.amazon.com/AmazonCloudWatch/latest/logs/AWS-logs-and-resource-policy.html
// [CreateDelivery]: https://docs.aws.amazon.com/AmazonCloudWatchLogs/latest/APIReference/API_CreateDelivery.html
// [PutDeliveryDestinationPolicy]: https://docs.aws.amazon.com/AmazonCloudWatchLogs/latest/APIReference/API_PutDeliveryDestinationPolicy.html
func (c *Client) PutDeliverySource(ctx context.Context, params *PutDeliverySourceInput, optFns ...func(*Options)) (*PutDeliverySourceOutput, error) {
	if params == nil {
		params = &PutDeliverySourceInput{}
	}

	result, metadata, err := c.invokeOperation(ctx, "PutDeliverySource", params, optFns, c.addOperationPutDeliverySourceMiddlewares)
	if err != nil {
		return nil, err
	}

	out := result.(*PutDeliverySourceOutput)
	out.ResultMetadata = metadata
	return out, nil
}

type PutDeliverySourceInput struct {

	// Defines the type of log that the source is sending.
	//
	//   - For Amazon CodeWhisperer, the valid value is EVENT_LOGS .
	//
	//   - For IAM Identity Centerr, the valid value is ERROR_LOGS .
	//
	//   - For Amazon WorkMail, the valid values are ACCESS_CONTROL_LOGS ,
	//   AUTHENTICATION_LOGS , WORKMAIL_AVAILABILITY_PROVIDER_LOGS , and
	//   WORKMAIL_MAILBOX_ACCESS_LOGS .
	//
	// This member is required.
	LogType *string

	// A name for this delivery source. This name must be unique for all delivery
	// sources in your account.
	//
	// This member is required.
	Name *string

	// The ARN of the Amazon Web Services resource that is generating and sending
	// logs. For example,
	// arn:aws:workmail:us-east-1:123456789012:organization/m-1234EXAMPLEabcd1234abcd1234abcd1234
	//
	// This member is required.
	ResourceArn *string

	// An optional list of key-value pairs to associate with the resource.
	//
	// For more information about tagging, see [Tagging Amazon Web Services resources]
	//
	// [Tagging Amazon Web Services resources]: https://docs.aws.amazon.com/general/latest/gr/aws_tagging.html
	Tags map[string]string

	noSmithyDocumentSerde
}

type PutDeliverySourceOutput struct {

	// A structure containing information about the delivery source that was just
	// created or updated.
	DeliverySource *types.DeliverySource

	// Metadata pertaining to the operation's result.
	ResultMetadata middleware.Metadata

	noSmithyDocumentSerde
}

func (c *Client) addOperationPutDeliverySourceMiddlewares(stack *middleware.Stack, options Options) (err error) {
	if err := stack.Serialize.Add(&setOperationInputMiddleware{}, middleware.After); err != nil {
		return err
	}
	err = stack.Serialize.Add(&awsAwsjson11_serializeOpPutDeliverySource{}, middleware.After)
	if err != nil {
		return err
	}
	err = stack.Deserialize.Add(&awsAwsjson11_deserializeOpPutDeliverySource{}, middleware.After)
	if err != nil {
		return err
	}
	if err := addProtocolFinalizerMiddlewares(stack, options, "PutDeliverySource"); err != nil {
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
	if err = addOpPutDeliverySourceValidationMiddleware(stack); err != nil {
		return err
	}
	if err = stack.Initialize.Add(newServiceMetadataMiddleware_opPutDeliverySource(options.Region), middleware.Before); err != nil {
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

func newServiceMetadataMiddleware_opPutDeliverySource(region string) *awsmiddleware.RegisterServiceMetadata {
	return &awsmiddleware.RegisterServiceMetadata{
		Region:        region,
		ServiceID:     ServiceID,
		OperationName: "PutDeliverySource",
	}
}
