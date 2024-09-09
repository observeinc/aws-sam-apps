// Code generated by smithy-go-codegen DO NOT EDIT.

package s3

import (
	"context"
	"fmt"
	awsmiddleware "github.com/aws/aws-sdk-go-v2/aws/middleware"
	"github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	s3cust "github.com/aws/aws-sdk-go-v2/service/s3/internal/customizations"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go/middleware"
	smithyhttp "github.com/aws/smithy-go/transport/http"
)

// This operation is not supported by directory buckets.
//
// Returns a list of all buckets owned by the authenticated sender of the request.
// To use this operation, you must have the s3:ListAllMyBuckets permission.
//
// For information about Amazon S3 buckets, see [Creating, configuring, and working with Amazon S3 buckets].
//
// [Creating, configuring, and working with Amazon S3 buckets]: https://docs.aws.amazon.com/AmazonS3/latest/userguide/creating-buckets-s3.html
func (c *Client) ListBuckets(ctx context.Context, params *ListBucketsInput, optFns ...func(*Options)) (*ListBucketsOutput, error) {
	if params == nil {
		params = &ListBucketsInput{}
	}

	result, metadata, err := c.invokeOperation(ctx, "ListBuckets", params, optFns, c.addOperationListBucketsMiddlewares)
	if err != nil {
		return nil, err
	}

	out := result.(*ListBucketsOutput)
	out.ResultMetadata = metadata
	return out, nil
}

type ListBucketsInput struct {

	// ContinuationToken indicates to Amazon S3 that the list is being continued on
	// this bucket with a token. ContinuationToken is obfuscated and is not a real
	// key. You can use this ContinuationToken for pagination of the list results.
	//
	// Length Constraints: Minimum length of 0. Maximum length of 1024.
	//
	// Required: No.
	ContinuationToken *string

	// Maximum number of buckets to be returned in response. When the number is more
	// than the count of buckets that are owned by an Amazon Web Services account,
	// return all the buckets in response.
	MaxBuckets *int32

	noSmithyDocumentSerde
}

type ListBucketsOutput struct {

	// The list of buckets owned by the requester.
	Buckets []types.Bucket

	// ContinuationToken is included in the response when there are more buckets that
	// can be listed with pagination. The next ListBuckets request to Amazon S3 can be
	// continued with this ContinuationToken . ContinuationToken is obfuscated and is
	// not a real bucket.
	ContinuationToken *string

	// The owner of the buckets listed.
	Owner *types.Owner

	// Metadata pertaining to the operation's result.
	ResultMetadata middleware.Metadata

	noSmithyDocumentSerde
}

func (c *Client) addOperationListBucketsMiddlewares(stack *middleware.Stack, options Options) (err error) {
	if err := stack.Serialize.Add(&setOperationInputMiddleware{}, middleware.After); err != nil {
		return err
	}
	err = stack.Serialize.Add(&awsRestxml_serializeOpListBuckets{}, middleware.After)
	if err != nil {
		return err
	}
	err = stack.Deserialize.Add(&awsRestxml_deserializeOpListBuckets{}, middleware.After)
	if err != nil {
		return err
	}
	if err := addProtocolFinalizerMiddlewares(stack, options, "ListBuckets"); err != nil {
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
	if err = addPutBucketContextMiddleware(stack); err != nil {
		return err
	}
	if err = addTimeOffsetBuild(stack, c); err != nil {
		return err
	}
	if err = addUserAgentRetryMode(stack, options); err != nil {
		return err
	}
	if err = addIsExpressUserAgent(stack); err != nil {
		return err
	}
	if err = stack.Initialize.Add(newServiceMetadataMiddleware_opListBuckets(options.Region), middleware.Before); err != nil {
		return err
	}
	if err = addMetadataRetrieverMiddleware(stack); err != nil {
		return err
	}
	if err = addRecursionDetection(stack); err != nil {
		return err
	}
	if err = addListBucketsUpdateEndpoint(stack, options); err != nil {
		return err
	}
	if err = addResponseErrorMiddleware(stack); err != nil {
		return err
	}
	if err = v4.AddContentSHA256HeaderMiddleware(stack); err != nil {
		return err
	}
	if err = disableAcceptEncodingGzip(stack); err != nil {
		return err
	}
	if err = addRequestResponseLogging(stack, options); err != nil {
		return err
	}
	if err = addDisableHTTPSMiddleware(stack, options); err != nil {
		return err
	}
	if err = addSerializeImmutableHostnameBucketMiddleware(stack, options); err != nil {
		return err
	}
	return nil
}

// ListBucketsPaginatorOptions is the paginator options for ListBuckets
type ListBucketsPaginatorOptions struct {
	// Maximum number of buckets to be returned in response. When the number is more
	// than the count of buckets that are owned by an Amazon Web Services account,
	// return all the buckets in response.
	Limit int32

	// Set to true if pagination should stop if the service returns a pagination token
	// that matches the most recent token provided to the service.
	StopOnDuplicateToken bool
}

// ListBucketsPaginator is a paginator for ListBuckets
type ListBucketsPaginator struct {
	options   ListBucketsPaginatorOptions
	client    ListBucketsAPIClient
	params    *ListBucketsInput
	nextToken *string
	firstPage bool
}

// NewListBucketsPaginator returns a new ListBucketsPaginator
func NewListBucketsPaginator(client ListBucketsAPIClient, params *ListBucketsInput, optFns ...func(*ListBucketsPaginatorOptions)) *ListBucketsPaginator {
	if params == nil {
		params = &ListBucketsInput{}
	}

	options := ListBucketsPaginatorOptions{}
	if params.MaxBuckets != nil {
		options.Limit = *params.MaxBuckets
	}

	for _, fn := range optFns {
		fn(&options)
	}

	return &ListBucketsPaginator{
		options:   options,
		client:    client,
		params:    params,
		firstPage: true,
		nextToken: params.ContinuationToken,
	}
}

// HasMorePages returns a boolean indicating whether more pages are available
func (p *ListBucketsPaginator) HasMorePages() bool {
	return p.firstPage || (p.nextToken != nil && len(*p.nextToken) != 0)
}

// NextPage retrieves the next ListBuckets page.
func (p *ListBucketsPaginator) NextPage(ctx context.Context, optFns ...func(*Options)) (*ListBucketsOutput, error) {
	if !p.HasMorePages() {
		return nil, fmt.Errorf("no more pages available")
	}

	params := *p.params
	params.ContinuationToken = p.nextToken

	var limit *int32
	if p.options.Limit > 0 {
		limit = &p.options.Limit
	}
	params.MaxBuckets = limit

	optFns = append([]func(*Options){
		addIsPaginatorUserAgent,
	}, optFns...)
	result, err := p.client.ListBuckets(ctx, &params, optFns...)
	if err != nil {
		return nil, err
	}
	p.firstPage = false

	prevToken := p.nextToken
	p.nextToken = result.ContinuationToken

	if p.options.StopOnDuplicateToken &&
		prevToken != nil &&
		p.nextToken != nil &&
		*prevToken == *p.nextToken {
		p.nextToken = nil
	}

	return result, nil
}

// ListBucketsAPIClient is a client that implements the ListBuckets operation.
type ListBucketsAPIClient interface {
	ListBuckets(context.Context, *ListBucketsInput, ...func(*Options)) (*ListBucketsOutput, error)
}

var _ ListBucketsAPIClient = (*Client)(nil)

func newServiceMetadataMiddleware_opListBuckets(region string) *awsmiddleware.RegisterServiceMetadata {
	return &awsmiddleware.RegisterServiceMetadata{
		Region:        region,
		ServiceID:     ServiceID,
		OperationName: "ListBuckets",
	}
}

func addListBucketsUpdateEndpoint(stack *middleware.Stack, options Options) error {
	return s3cust.UpdateEndpoint(stack, s3cust.UpdateEndpointOptions{
		Accessor: s3cust.UpdateEndpointParameterAccessor{
			GetBucketFromInput: nopGetBucketAccessor,
		},
		UsePathStyle:                   options.UsePathStyle,
		UseAccelerate:                  options.UseAccelerate,
		SupportsAccelerate:             false,
		TargetS3ObjectLambda:           false,
		EndpointResolver:               options.EndpointResolver,
		EndpointResolverOptions:        options.EndpointOptions,
		UseARNRegion:                   options.UseARNRegion,
		DisableMultiRegionAccessPoints: options.DisableMultiRegionAccessPoints,
	})
}
