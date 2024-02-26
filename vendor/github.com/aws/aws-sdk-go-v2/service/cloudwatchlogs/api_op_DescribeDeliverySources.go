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

// Retrieves a list of the delivery sources that have been created in the account.
func (c *Client) DescribeDeliverySources(ctx context.Context, params *DescribeDeliverySourcesInput, optFns ...func(*Options)) (*DescribeDeliverySourcesOutput, error) {
	if params == nil {
		params = &DescribeDeliverySourcesInput{}
	}

	result, metadata, err := c.invokeOperation(ctx, "DescribeDeliverySources", params, optFns, c.addOperationDescribeDeliverySourcesMiddlewares)
	if err != nil {
		return nil, err
	}

	out := result.(*DescribeDeliverySourcesOutput)
	out.ResultMetadata = metadata
	return out, nil
}

type DescribeDeliverySourcesInput struct {

	// Optionally specify the maximum number of delivery sources to return in the
	// response.
	Limit *int32

	// The token for the next set of items to return. The token expires after 24 hours.
	NextToken *string

	noSmithyDocumentSerde
}

type DescribeDeliverySourcesOutput struct {

	// An array of structures. Each structure contains information about one delivery
	// source in the account.
	DeliverySources []types.DeliverySource

	// The token for the next set of items to return. The token expires after 24 hours.
	NextToken *string

	// Metadata pertaining to the operation's result.
	ResultMetadata middleware.Metadata

	noSmithyDocumentSerde
}

func (c *Client) addOperationDescribeDeliverySourcesMiddlewares(stack *middleware.Stack, options Options) (err error) {
	if err := stack.Serialize.Add(&setOperationInputMiddleware{}, middleware.After); err != nil {
		return err
	}
	err = stack.Serialize.Add(&awsAwsjson11_serializeOpDescribeDeliverySources{}, middleware.After)
	if err != nil {
		return err
	}
	err = stack.Deserialize.Add(&awsAwsjson11_deserializeOpDescribeDeliverySources{}, middleware.After)
	if err != nil {
		return err
	}
	if err := addProtocolFinalizerMiddlewares(stack, options, "DescribeDeliverySources"); err != nil {
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
	if err = stack.Initialize.Add(newServiceMetadataMiddleware_opDescribeDeliverySources(options.Region), middleware.Before); err != nil {
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

// DescribeDeliverySourcesAPIClient is a client that implements the
// DescribeDeliverySources operation.
type DescribeDeliverySourcesAPIClient interface {
	DescribeDeliverySources(context.Context, *DescribeDeliverySourcesInput, ...func(*Options)) (*DescribeDeliverySourcesOutput, error)
}

var _ DescribeDeliverySourcesAPIClient = (*Client)(nil)

// DescribeDeliverySourcesPaginatorOptions is the paginator options for
// DescribeDeliverySources
type DescribeDeliverySourcesPaginatorOptions struct {
	// Optionally specify the maximum number of delivery sources to return in the
	// response.
	Limit int32

	// Set to true if pagination should stop if the service returns a pagination token
	// that matches the most recent token provided to the service.
	StopOnDuplicateToken bool
}

// DescribeDeliverySourcesPaginator is a paginator for DescribeDeliverySources
type DescribeDeliverySourcesPaginator struct {
	options   DescribeDeliverySourcesPaginatorOptions
	client    DescribeDeliverySourcesAPIClient
	params    *DescribeDeliverySourcesInput
	nextToken *string
	firstPage bool
}

// NewDescribeDeliverySourcesPaginator returns a new
// DescribeDeliverySourcesPaginator
func NewDescribeDeliverySourcesPaginator(client DescribeDeliverySourcesAPIClient, params *DescribeDeliverySourcesInput, optFns ...func(*DescribeDeliverySourcesPaginatorOptions)) *DescribeDeliverySourcesPaginator {
	if params == nil {
		params = &DescribeDeliverySourcesInput{}
	}

	options := DescribeDeliverySourcesPaginatorOptions{}
	if params.Limit != nil {
		options.Limit = *params.Limit
	}

	for _, fn := range optFns {
		fn(&options)
	}

	return &DescribeDeliverySourcesPaginator{
		options:   options,
		client:    client,
		params:    params,
		firstPage: true,
		nextToken: params.NextToken,
	}
}

// HasMorePages returns a boolean indicating whether more pages are available
func (p *DescribeDeliverySourcesPaginator) HasMorePages() bool {
	return p.firstPage || (p.nextToken != nil && len(*p.nextToken) != 0)
}

// NextPage retrieves the next DescribeDeliverySources page.
func (p *DescribeDeliverySourcesPaginator) NextPage(ctx context.Context, optFns ...func(*Options)) (*DescribeDeliverySourcesOutput, error) {
	if !p.HasMorePages() {
		return nil, fmt.Errorf("no more pages available")
	}

	params := *p.params
	params.NextToken = p.nextToken

	var limit *int32
	if p.options.Limit > 0 {
		limit = &p.options.Limit
	}
	params.Limit = limit

	result, err := p.client.DescribeDeliverySources(ctx, &params, optFns...)
	if err != nil {
		return nil, err
	}
	p.firstPage = false

	prevToken := p.nextToken
	p.nextToken = result.NextToken

	if p.options.StopOnDuplicateToken &&
		prevToken != nil &&
		p.nextToken != nil &&
		*prevToken == *p.nextToken {
		p.nextToken = nil
	}

	return result, nil
}

func newServiceMetadataMiddleware_opDescribeDeliverySources(region string) *awsmiddleware.RegisterServiceMetadata {
	return &awsmiddleware.RegisterServiceMetadata{
		Region:        region,
		ServiceID:     ServiceID,
		OperationName: "DescribeDeliverySources",
	}
}
