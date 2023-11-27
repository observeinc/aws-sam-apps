// Code generated by smithy-go-codegen DO NOT EDIT.

package cloudwatchlogs

import (
	"context"
	"fmt"
	awsmiddleware "github.com/aws/aws-sdk-go-v2/aws/middleware"
	"github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
	"github.com/aws/smithy-go/middleware"
	smithyhttp "github.com/aws/smithy-go/transport/http"
)

// Lists the specified log groups. You can list all your log groups or filter the
// results by prefix. The results are ASCII-sorted by log group name. CloudWatch
// Logs doesn’t support IAM policies that control access to the DescribeLogGroups
// action by using the aws:ResourceTag/key-name  condition key. Other CloudWatch
// Logs actions do support the use of the aws:ResourceTag/key-name  condition key
// to control access. For more information about using tags to control access, see
// Controlling access to Amazon Web Services resources using tags (https://docs.aws.amazon.com/IAM/latest/UserGuide/access_tags.html)
// . If you are using CloudWatch cross-account observability, you can use this
// operation in a monitoring account and view data from the linked source accounts.
// For more information, see CloudWatch cross-account observability (https://docs.aws.amazon.com/AmazonCloudWatch/latest/monitoring/CloudWatch-Unified-Cross-Account.html)
// .
func (c *Client) DescribeLogGroups(ctx context.Context, params *DescribeLogGroupsInput, optFns ...func(*Options)) (*DescribeLogGroupsOutput, error) {
	if params == nil {
		params = &DescribeLogGroupsInput{}
	}

	result, metadata, err := c.invokeOperation(ctx, "DescribeLogGroups", params, optFns, c.addOperationDescribeLogGroupsMiddlewares)
	if err != nil {
		return nil, err
	}

	out := result.(*DescribeLogGroupsOutput)
	out.ResultMetadata = metadata
	return out, nil
}

type DescribeLogGroupsInput struct {

	// When includeLinkedAccounts is set to True , use this parameter to specify the
	// list of accounts to search. You can specify as many as 20 account IDs in the
	// array.
	AccountIdentifiers []string

	// If you are using a monitoring account, set this to True to have the operation
	// return log groups in the accounts listed in accountIdentifiers . If this
	// parameter is set to true and accountIdentifiers contains a null value, the
	// operation returns all log groups in the monitoring account and all log groups in
	// all source accounts that are linked to the monitoring account.
	IncludeLinkedAccounts *bool

	// The maximum number of items returned. If you don't specify a value, the default
	// is up to 50 items.
	Limit *int32

	// Specifies the log group class for this log group. There are two classes:
	//   - The Standard log class supports all CloudWatch Logs features.
	//   - The Infrequent Access log class supports a subset of CloudWatch Logs
	//   features and incurs lower costs.
	// For details about the features supported by each class, see Log classes (https://docs.aws.amazon.com/AmazonCloudWatch/latest/logs/CloudWatch_Logs_Log_Classes.html)
	LogGroupClass types.LogGroupClass

	// If you specify a string for this parameter, the operation returns only log
	// groups that have names that match the string based on a case-sensitive substring
	// search. For example, if you specify Foo , log groups named FooBar , aws/Foo ,
	// and GroupFoo would match, but foo , F/o/o and Froo would not match. If you
	// specify logGroupNamePattern in your request, then only arn , creationTime , and
	// logGroupName are included in the response. logGroupNamePattern and
	// logGroupNamePrefix are mutually exclusive. Only one of these parameters can be
	// passed.
	LogGroupNamePattern *string

	// The prefix to match. logGroupNamePrefix and logGroupNamePattern are mutually
	// exclusive. Only one of these parameters can be passed.
	LogGroupNamePrefix *string

	// The token for the next set of items to return. (You received this token from a
	// previous call.)
	NextToken *string

	noSmithyDocumentSerde
}

type DescribeLogGroupsOutput struct {

	// The log groups. If the retentionInDays value is not included for a log group,
	// then that log group's events do not expire.
	LogGroups []types.LogGroup

	// The token for the next set of items to return. The token expires after 24 hours.
	NextToken *string

	// Metadata pertaining to the operation's result.
	ResultMetadata middleware.Metadata

	noSmithyDocumentSerde
}

func (c *Client) addOperationDescribeLogGroupsMiddlewares(stack *middleware.Stack, options Options) (err error) {
	if err := stack.Serialize.Add(&setOperationInputMiddleware{}, middleware.After); err != nil {
		return err
	}
	err = stack.Serialize.Add(&awsAwsjson11_serializeOpDescribeLogGroups{}, middleware.After)
	if err != nil {
		return err
	}
	err = stack.Deserialize.Add(&awsAwsjson11_deserializeOpDescribeLogGroups{}, middleware.After)
	if err != nil {
		return err
	}
	if err := addProtocolFinalizerMiddlewares(stack, options, "DescribeLogGroups"); err != nil {
		return fmt.Errorf("add protocol finalizers: %v", err)
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
	if err = addSetLegacyContextSigningOptionsMiddleware(stack); err != nil {
		return err
	}
	if err = stack.Initialize.Add(newServiceMetadataMiddleware_opDescribeLogGroups(options.Region), middleware.Before); err != nil {
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
	if err = addDisableHTTPSMiddleware(stack, options); err != nil {
		return err
	}
	return nil
}

// DescribeLogGroupsAPIClient is a client that implements the DescribeLogGroups
// operation.
type DescribeLogGroupsAPIClient interface {
	DescribeLogGroups(context.Context, *DescribeLogGroupsInput, ...func(*Options)) (*DescribeLogGroupsOutput, error)
}

var _ DescribeLogGroupsAPIClient = (*Client)(nil)

// DescribeLogGroupsPaginatorOptions is the paginator options for DescribeLogGroups
type DescribeLogGroupsPaginatorOptions struct {
	// The maximum number of items returned. If you don't specify a value, the default
	// is up to 50 items.
	Limit int32

	// Set to true if pagination should stop if the service returns a pagination token
	// that matches the most recent token provided to the service.
	StopOnDuplicateToken bool
}

// DescribeLogGroupsPaginator is a paginator for DescribeLogGroups
type DescribeLogGroupsPaginator struct {
	options   DescribeLogGroupsPaginatorOptions
	client    DescribeLogGroupsAPIClient
	params    *DescribeLogGroupsInput
	nextToken *string
	firstPage bool
}

// NewDescribeLogGroupsPaginator returns a new DescribeLogGroupsPaginator
func NewDescribeLogGroupsPaginator(client DescribeLogGroupsAPIClient, params *DescribeLogGroupsInput, optFns ...func(*DescribeLogGroupsPaginatorOptions)) *DescribeLogGroupsPaginator {
	if params == nil {
		params = &DescribeLogGroupsInput{}
	}

	options := DescribeLogGroupsPaginatorOptions{}
	if params.Limit != nil {
		options.Limit = *params.Limit
	}

	for _, fn := range optFns {
		fn(&options)
	}

	return &DescribeLogGroupsPaginator{
		options:   options,
		client:    client,
		params:    params,
		firstPage: true,
		nextToken: params.NextToken,
	}
}

// HasMorePages returns a boolean indicating whether more pages are available
func (p *DescribeLogGroupsPaginator) HasMorePages() bool {
	return p.firstPage || (p.nextToken != nil && len(*p.nextToken) != 0)
}

// NextPage retrieves the next DescribeLogGroups page.
func (p *DescribeLogGroupsPaginator) NextPage(ctx context.Context, optFns ...func(*Options)) (*DescribeLogGroupsOutput, error) {
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

	result, err := p.client.DescribeLogGroups(ctx, &params, optFns...)
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

func newServiceMetadataMiddleware_opDescribeLogGroups(region string) *awsmiddleware.RegisterServiceMetadata {
	return &awsmiddleware.RegisterServiceMetadata{
		Region:        region,
		ServiceID:     ServiceID,
		OperationName: "DescribeLogGroups",
	}
}
