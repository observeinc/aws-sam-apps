// Code generated by smithy-go-codegen DO NOT EDIT.

package cloudwatchlogs

import (
	"context"
	"fmt"
	awsmiddleware "github.com/aws/aws-sdk-go-v2/aws/middleware"
	"github.com/aws/smithy-go/middleware"
	smithyhttp "github.com/aws/smithy-go/transport/http"
)

// Creates or updates a query definition for CloudWatch Logs Insights. For more
// information, see [Analyzing Log Data with CloudWatch Logs Insights].
//
// To update a query definition, specify its queryDefinitionId in your request.
// The values of name , queryString , and logGroupNames are changed to the values
// that you specify in your update operation. No current values are retained from
// the current query definition. For example, imagine updating a current query
// definition that includes log groups. If you don't specify the logGroupNames
// parameter in your update operation, the query definition changes to contain no
// log groups.
//
// You must have the logs:PutQueryDefinition permission to be able to perform this
// operation.
//
// [Analyzing Log Data with CloudWatch Logs Insights]: https://docs.aws.amazon.com/AmazonCloudWatch/latest/logs/AnalyzingLogData.html
func (c *Client) PutQueryDefinition(ctx context.Context, params *PutQueryDefinitionInput, optFns ...func(*Options)) (*PutQueryDefinitionOutput, error) {
	if params == nil {
		params = &PutQueryDefinitionInput{}
	}

	result, metadata, err := c.invokeOperation(ctx, "PutQueryDefinition", params, optFns, c.addOperationPutQueryDefinitionMiddlewares)
	if err != nil {
		return nil, err
	}

	out := result.(*PutQueryDefinitionOutput)
	out.ResultMetadata = metadata
	return out, nil
}

type PutQueryDefinitionInput struct {

	// A name for the query definition. If you are saving numerous query definitions,
	// we recommend that you name them. This way, you can find the ones you want by
	// using the first part of the name as a filter in the queryDefinitionNamePrefix
	// parameter of [DescribeQueryDefinitions].
	//
	// [DescribeQueryDefinitions]: https://docs.aws.amazon.com/AmazonCloudWatchLogs/latest/APIReference/API_DescribeQueryDefinitions.html
	//
	// This member is required.
	Name *string

	// The query string to use for this definition. For more information, see [CloudWatch Logs Insights Query Syntax].
	//
	// [CloudWatch Logs Insights Query Syntax]: https://docs.aws.amazon.com/AmazonCloudWatch/latest/logs/CWL_QuerySyntax.html
	//
	// This member is required.
	QueryString *string

	// Used as an idempotency token, to avoid returning an exception if the service
	// receives the same request twice because of a network
	//
	// error.
	ClientToken *string

	// Use this parameter to include specific log groups as part of your query
	// definition.
	//
	// If you are updating a query definition and you omit this parameter, then the
	// updated definition will contain no log groups.
	LogGroupNames []string

	// If you are updating a query definition, use this parameter to specify the ID of
	// the query definition that you want to update. You can use [DescribeQueryDefinitions]to retrieve the IDs
	// of your saved query definitions.
	//
	// If you are creating a query definition, do not specify this parameter.
	// CloudWatch generates a unique ID for the new query definition and include it in
	// the response to this operation.
	//
	// [DescribeQueryDefinitions]: https://docs.aws.amazon.com/AmazonCloudWatchLogs/latest/APIReference/API_DescribeQueryDefinitions.html
	QueryDefinitionId *string

	noSmithyDocumentSerde
}

type PutQueryDefinitionOutput struct {

	// The ID of the query definition.
	QueryDefinitionId *string

	// Metadata pertaining to the operation's result.
	ResultMetadata middleware.Metadata

	noSmithyDocumentSerde
}

func (c *Client) addOperationPutQueryDefinitionMiddlewares(stack *middleware.Stack, options Options) (err error) {
	if err := stack.Serialize.Add(&setOperationInputMiddleware{}, middleware.After); err != nil {
		return err
	}
	err = stack.Serialize.Add(&awsAwsjson11_serializeOpPutQueryDefinition{}, middleware.After)
	if err != nil {
		return err
	}
	err = stack.Deserialize.Add(&awsAwsjson11_deserializeOpPutQueryDefinition{}, middleware.After)
	if err != nil {
		return err
	}
	if err := addProtocolFinalizerMiddlewares(stack, options, "PutQueryDefinition"); err != nil {
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
	if err = addSpanRetryLoop(stack, options); err != nil {
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
	if err = addIdempotencyToken_opPutQueryDefinitionMiddleware(stack, options); err != nil {
		return err
	}
	if err = addOpPutQueryDefinitionValidationMiddleware(stack); err != nil {
		return err
	}
	if err = stack.Initialize.Add(newServiceMetadataMiddleware_opPutQueryDefinition(options.Region), middleware.Before); err != nil {
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
	if err = addSpanInitializeStart(stack); err != nil {
		return err
	}
	if err = addSpanInitializeEnd(stack); err != nil {
		return err
	}
	if err = addSpanBuildRequestStart(stack); err != nil {
		return err
	}
	if err = addSpanBuildRequestEnd(stack); err != nil {
		return err
	}
	return nil
}

type idempotencyToken_initializeOpPutQueryDefinition struct {
	tokenProvider IdempotencyTokenProvider
}

func (*idempotencyToken_initializeOpPutQueryDefinition) ID() string {
	return "OperationIdempotencyTokenAutoFill"
}

func (m *idempotencyToken_initializeOpPutQueryDefinition) HandleInitialize(ctx context.Context, in middleware.InitializeInput, next middleware.InitializeHandler) (
	out middleware.InitializeOutput, metadata middleware.Metadata, err error,
) {
	if m.tokenProvider == nil {
		return next.HandleInitialize(ctx, in)
	}

	input, ok := in.Parameters.(*PutQueryDefinitionInput)
	if !ok {
		return out, metadata, fmt.Errorf("expected middleware input to be of type *PutQueryDefinitionInput ")
	}

	if input.ClientToken == nil {
		t, err := m.tokenProvider.GetIdempotencyToken()
		if err != nil {
			return out, metadata, err
		}
		input.ClientToken = &t
	}
	return next.HandleInitialize(ctx, in)
}
func addIdempotencyToken_opPutQueryDefinitionMiddleware(stack *middleware.Stack, cfg Options) error {
	return stack.Initialize.Add(&idempotencyToken_initializeOpPutQueryDefinition{tokenProvider: cfg.IdempotencyTokenProvider}, middleware.Before)
}

func newServiceMetadataMiddleware_opPutQueryDefinition(region string) *awsmiddleware.RegisterServiceMetadata {
	return &awsmiddleware.RegisterServiceMetadata{
		Region:        region,
		ServiceID:     ServiceID,
		OperationName: "PutQueryDefinition",
	}
}
