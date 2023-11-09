// Code generated by smithy-go-codegen DO NOT EDIT.

package sqs

import (
	"context"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsmiddleware "github.com/aws/aws-sdk-go-v2/aws/middleware"
	"github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	internalauth "github.com/aws/aws-sdk-go-v2/internal/auth"
	smithyendpoints "github.com/aws/smithy-go/endpoints"
	"github.com/aws/smithy-go/middleware"
	smithyhttp "github.com/aws/smithy-go/transport/http"
)

// Changes the visibility timeout of a specified message in a queue to a new
// value. The default visibility timeout for a message is 30 seconds. The minimum
// is 0 seconds. The maximum is 12 hours. For more information, see Visibility
// Timeout (https://docs.aws.amazon.com/AWSSimpleQueueService/latest/SQSDeveloperGuide/sqs-visibility-timeout.html)
// in the Amazon SQS Developer Guide. For example, if the default timeout for a
// queue is 60 seconds, 15 seconds have elapsed since you received the message, and
// you send a ChangeMessageVisibility call with VisibilityTimeout set to 10
// seconds, the 10 seconds begin to count from the time that you make the
// ChangeMessageVisibility call. Thus, any attempt to change the visibility timeout
// or to delete that message 10 seconds after you initially change the visibility
// timeout (a total of 25 seconds) might result in an error. An Amazon SQS message
// has three basic states:
//   - Sent to a queue by a producer.
//   - Received from the queue by a consumer.
//   - Deleted from the queue.
//
// A message is considered to be stored after it is sent to a queue by a producer,
// but not yet received from the queue by a consumer (that is, between states 1 and
// 2). There is no limit to the number of stored messages. A message is considered
// to be in flight after it is received from a queue by a consumer, but not yet
// deleted from the queue (that is, between states 2 and 3). There is a limit to
// the number of in flight messages. Limits that apply to in flight messages are
// unrelated to the unlimited number of stored messages. For most standard queues
// (depending on queue traffic and message backlog), there can be a maximum of
// approximately 120,000 in flight messages (received from a queue by a consumer,
// but not yet deleted from the queue). If you reach this limit, Amazon SQS returns
// the OverLimit error message. To avoid reaching the limit, you should delete
// messages from the queue after they're processed. You can also increase the
// number of queues you use to process your messages. To request a limit increase,
// file a support request (https://console.aws.amazon.com/support/home#/case/create?issueType=service-limit-increase&limitType=service-code-sqs)
// . For FIFO queues, there can be a maximum of 20,000 in flight messages (received
// from a queue by a consumer, but not yet deleted from the queue). If you reach
// this limit, Amazon SQS returns no error messages. If you attempt to set the
// VisibilityTimeout to a value greater than the maximum time left, Amazon SQS
// returns an error. Amazon SQS doesn't automatically recalculate and increase the
// timeout to the maximum remaining time. Unlike with a queue, when you change the
// visibility timeout for a specific message the timeout value is applied
// immediately but isn't saved in memory for that message. If you don't delete a
// message after it is received, the visibility timeout for the message reverts to
// the original timeout value (not to the value you set using the
// ChangeMessageVisibility action) the next time the message is received.
func (c *Client) ChangeMessageVisibility(ctx context.Context, params *ChangeMessageVisibilityInput, optFns ...func(*Options)) (*ChangeMessageVisibilityOutput, error) {
	if params == nil {
		params = &ChangeMessageVisibilityInput{}
	}

	result, metadata, err := c.invokeOperation(ctx, "ChangeMessageVisibility", params, optFns, c.addOperationChangeMessageVisibilityMiddlewares)
	if err != nil {
		return nil, err
	}

	out := result.(*ChangeMessageVisibilityOutput)
	out.ResultMetadata = metadata
	return out, nil
}

type ChangeMessageVisibilityInput struct {

	// The URL of the Amazon SQS queue whose message's visibility is changed. Queue
	// URLs and names are case-sensitive.
	//
	// This member is required.
	QueueUrl *string

	// The receipt handle associated with the message, whose visibility timeout is
	// changed. This parameter is returned by the ReceiveMessage action.
	//
	// This member is required.
	ReceiptHandle *string

	// The new value for the message's visibility timeout (in seconds). Values range: 0
	// to 43200 . Maximum: 12 hours.
	//
	// This member is required.
	VisibilityTimeout int32

	noSmithyDocumentSerde
}

type ChangeMessageVisibilityOutput struct {
	// Metadata pertaining to the operation's result.
	ResultMetadata middleware.Metadata

	noSmithyDocumentSerde
}

func (c *Client) addOperationChangeMessageVisibilityMiddlewares(stack *middleware.Stack, options Options) (err error) {
	err = stack.Serialize.Add(&awsAwsjson10_serializeOpChangeMessageVisibility{}, middleware.After)
	if err != nil {
		return err
	}
	err = stack.Deserialize.Add(&awsAwsjson10_deserializeOpChangeMessageVisibility{}, middleware.After)
	if err != nil {
		return err
	}
	if err = addlegacyEndpointContextSetter(stack, options); err != nil {
		return err
	}
	if err = addSetLoggerMiddleware(stack, options); err != nil {
		return err
	}
	if err = awsmiddleware.AddClientRequestIDMiddleware(stack); err != nil {
		return err
	}
	if err = smithyhttp.AddComputeContentLengthMiddleware(stack); err != nil {
		return err
	}
	if err = addResolveEndpointMiddleware(stack, options); err != nil {
		return err
	}
	if err = v4.AddComputePayloadSHA256Middleware(stack); err != nil {
		return err
	}
	if err = addRetryMiddlewares(stack, options); err != nil {
		return err
	}
	if err = addHTTPSignerV4Middleware(stack, options); err != nil {
		return err
	}
	if err = awsmiddleware.AddRawResponseToMetadata(stack); err != nil {
		return err
	}
	if err = awsmiddleware.AddRecordResponseTiming(stack); err != nil {
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
	if err = addChangeMessageVisibilityResolveEndpointMiddleware(stack, options); err != nil {
		return err
	}
	if err = addOpChangeMessageVisibilityValidationMiddleware(stack); err != nil {
		return err
	}
	if err = stack.Initialize.Add(newServiceMetadataMiddleware_opChangeMessageVisibility(options.Region), middleware.Before); err != nil {
		return err
	}
	if err = awsmiddleware.AddRecursionDetection(stack); err != nil {
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
	if err = addendpointDisableHTTPSMiddleware(stack, options); err != nil {
		return err
	}
	return nil
}

func newServiceMetadataMiddleware_opChangeMessageVisibility(region string) *awsmiddleware.RegisterServiceMetadata {
	return &awsmiddleware.RegisterServiceMetadata{
		Region:        region,
		ServiceID:     ServiceID,
		SigningName:   "sqs",
		OperationName: "ChangeMessageVisibility",
	}
}

type opChangeMessageVisibilityResolveEndpointMiddleware struct {
	EndpointResolver EndpointResolverV2
	BuiltInResolver  builtInParameterResolver
}

func (*opChangeMessageVisibilityResolveEndpointMiddleware) ID() string {
	return "ResolveEndpointV2"
}

func (m *opChangeMessageVisibilityResolveEndpointMiddleware) HandleSerialize(ctx context.Context, in middleware.SerializeInput, next middleware.SerializeHandler) (
	out middleware.SerializeOutput, metadata middleware.Metadata, err error,
) {
	if awsmiddleware.GetRequiresLegacyEndpoints(ctx) {
		return next.HandleSerialize(ctx, in)
	}

	req, ok := in.Request.(*smithyhttp.Request)
	if !ok {
		return out, metadata, fmt.Errorf("unknown transport type %T", in.Request)
	}

	if m.EndpointResolver == nil {
		return out, metadata, fmt.Errorf("expected endpoint resolver to not be nil")
	}

	params := EndpointParameters{}

	m.BuiltInResolver.ResolveBuiltIns(&params)

	var resolvedEndpoint smithyendpoints.Endpoint
	resolvedEndpoint, err = m.EndpointResolver.ResolveEndpoint(ctx, params)
	if err != nil {
		return out, metadata, fmt.Errorf("failed to resolve service endpoint, %w", err)
	}

	req.URL = &resolvedEndpoint.URI

	for k := range resolvedEndpoint.Headers {
		req.Header.Set(
			k,
			resolvedEndpoint.Headers.Get(k),
		)
	}

	authSchemes, err := internalauth.GetAuthenticationSchemes(&resolvedEndpoint.Properties)
	if err != nil {
		var nfe *internalauth.NoAuthenticationSchemesFoundError
		if errors.As(err, &nfe) {
			// if no auth scheme is found, default to sigv4
			signingName := "sqs"
			signingRegion := m.BuiltInResolver.(*builtInResolver).Region
			ctx = awsmiddleware.SetSigningName(ctx, signingName)
			ctx = awsmiddleware.SetSigningRegion(ctx, signingRegion)

		}
		var ue *internalauth.UnSupportedAuthenticationSchemeSpecifiedError
		if errors.As(err, &ue) {
			return out, metadata, fmt.Errorf(
				"This operation requests signer version(s) %v but the client only supports %v",
				ue.UnsupportedSchemes,
				internalauth.SupportedSchemes,
			)
		}
	}

	for _, authScheme := range authSchemes {
		switch authScheme.(type) {
		case *internalauth.AuthenticationSchemeV4:
			v4Scheme, _ := authScheme.(*internalauth.AuthenticationSchemeV4)
			var signingName, signingRegion string
			if v4Scheme.SigningName == nil {
				signingName = "sqs"
			} else {
				signingName = *v4Scheme.SigningName
			}
			if v4Scheme.SigningRegion == nil {
				signingRegion = m.BuiltInResolver.(*builtInResolver).Region
			} else {
				signingRegion = *v4Scheme.SigningRegion
			}
			if v4Scheme.DisableDoubleEncoding != nil {
				// The signer sets an equivalent value at client initialization time.
				// Setting this context value will cause the signer to extract it
				// and override the value set at client initialization time.
				ctx = internalauth.SetDisableDoubleEncoding(ctx, *v4Scheme.DisableDoubleEncoding)
			}
			ctx = awsmiddleware.SetSigningName(ctx, signingName)
			ctx = awsmiddleware.SetSigningRegion(ctx, signingRegion)
			break
		case *internalauth.AuthenticationSchemeV4A:
			v4aScheme, _ := authScheme.(*internalauth.AuthenticationSchemeV4A)
			if v4aScheme.SigningName == nil {
				v4aScheme.SigningName = aws.String("sqs")
			}
			if v4aScheme.DisableDoubleEncoding != nil {
				// The signer sets an equivalent value at client initialization time.
				// Setting this context value will cause the signer to extract it
				// and override the value set at client initialization time.
				ctx = internalauth.SetDisableDoubleEncoding(ctx, *v4aScheme.DisableDoubleEncoding)
			}
			ctx = awsmiddleware.SetSigningName(ctx, *v4aScheme.SigningName)
			ctx = awsmiddleware.SetSigningRegion(ctx, v4aScheme.SigningRegionSet[0])
			break
		case *internalauth.AuthenticationSchemeNone:
			break
		}
	}

	return next.HandleSerialize(ctx, in)
}

func addChangeMessageVisibilityResolveEndpointMiddleware(stack *middleware.Stack, options Options) error {
	return stack.Serialize.Insert(&opChangeMessageVisibilityResolveEndpointMiddleware{
		EndpointResolver: options.EndpointResolverV2,
		BuiltInResolver: &builtInResolver{
			Region:       options.Region,
			UseDualStack: options.EndpointOptions.UseDualStackEndpoint,
			UseFIPS:      options.EndpointOptions.UseFIPSEndpoint,
			Endpoint:     options.BaseEndpoint,
		},
	}, "ResolveEndpoint", middleware.After)
}
