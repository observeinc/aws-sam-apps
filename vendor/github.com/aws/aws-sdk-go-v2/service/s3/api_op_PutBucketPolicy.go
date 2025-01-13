// Code generated by smithy-go-codegen DO NOT EDIT.

package s3

import (
	"context"
	"fmt"
	awsmiddleware "github.com/aws/aws-sdk-go-v2/aws/middleware"
	"github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	internalChecksum "github.com/aws/aws-sdk-go-v2/service/internal/checksum"
	s3cust "github.com/aws/aws-sdk-go-v2/service/s3/internal/customizations"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go/middleware"
	"github.com/aws/smithy-go/ptr"
	smithyhttp "github.com/aws/smithy-go/transport/http"
)

// Applies an Amazon S3 bucket policy to an Amazon S3 bucket.
//
// Directory buckets - For directory buckets, you must make requests for this API
// operation to the Regional endpoint. These endpoints support path-style requests
// in the format https://s3express-control.region-code.amazonaws.com/bucket-name .
// Virtual-hosted-style requests aren't supported. For more information about
// endpoints in Availability Zones, see [Regional and Zonal endpoints for directory buckets in Availability Zones]in the Amazon S3 User Guide. For more
// information about endpoints in Local Zones, see [Concepts for directory buckets in Local Zones]in the Amazon S3 User Guide.
//
// Permissions If you are using an identity other than the root user of the Amazon
// Web Services account that owns the bucket, the calling identity must both have
// the PutBucketPolicy permissions on the specified bucket and belong to the
// bucket owner's account in order to use this operation.
//
// If you don't have PutBucketPolicy permissions, Amazon S3 returns a 403 Access
// Denied error. If you have the correct permissions, but you're not using an
// identity that belongs to the bucket owner's account, Amazon S3 returns a 405
// Method Not Allowed error.
//
// To ensure that bucket owners don't inadvertently lock themselves out of their
// own buckets, the root principal in a bucket owner's Amazon Web Services account
// can perform the GetBucketPolicy , PutBucketPolicy , and DeleteBucketPolicy API
// actions, even if their bucket policy explicitly denies the root principal's
// access. Bucket owner root principals can only be blocked from performing these
// API actions by VPC endpoint policies and Amazon Web Services Organizations
// policies.
//
//   - General purpose bucket permissions - The s3:PutBucketPolicy permission is
//     required in a policy. For more information about general purpose buckets bucket
//     policies, see [Using Bucket Policies and User Policies]in the Amazon S3 User Guide.
//
//   - Directory bucket permissions - To grant access to this API operation, you
//     must have the s3express:PutBucketPolicy permission in an IAM identity-based
//     policy instead of a bucket policy. Cross-account access to this API operation
//     isn't supported. This operation can only be performed by the Amazon Web Services
//     account that owns the resource. For more information about directory bucket
//     policies and permissions, see [Amazon Web Services Identity and Access Management (IAM) for S3 Express One Zone]in the Amazon S3 User Guide.
//
// Example bucket policies  General purpose buckets example bucket policies - See [Bucket policy examples]
// in the Amazon S3 User Guide.
//
// Directory bucket example bucket policies - See [Example bucket policies for S3 Express One Zone] in the Amazon S3 User Guide.
//
// HTTP Host header syntax  Directory buckets - The HTTP Host header syntax is
// s3express-control.region-code.amazonaws.com .
//
// The following operations are related to PutBucketPolicy :
//
// [CreateBucket]
//
// [DeleteBucket]
//
// [Bucket policy examples]: https://docs.aws.amazon.com/AmazonS3/latest/userguide/example-bucket-policies.html
// [Concepts for directory buckets in Local Zones]: https://docs.aws.amazon.com/AmazonS3/latest/userguide/s3-lzs-for-directory-buckets.html
// [Example bucket policies for S3 Express One Zone]: https://docs.aws.amazon.com/AmazonS3/latest/userguide/s3-express-security-iam-example-bucket-policies.html
// [DeleteBucket]: https://docs.aws.amazon.com/AmazonS3/latest/API/API_DeleteBucket.html
// [Using Bucket Policies and User Policies]: https://docs.aws.amazon.com/AmazonS3/latest/dev/using-iam-policies.html
// [CreateBucket]: https://docs.aws.amazon.com/AmazonS3/latest/API/API_CreateBucket.html
// [Regional and Zonal endpoints for directory buckets in Availability Zones]: https://docs.aws.amazon.com/AmazonS3/latest/userguide/endpoint-directory-buckets-AZ.html
// [Amazon Web Services Identity and Access Management (IAM) for S3 Express One Zone]: https://docs.aws.amazon.com/AmazonS3/latest/userguide/s3-express-security-iam.html
func (c *Client) PutBucketPolicy(ctx context.Context, params *PutBucketPolicyInput, optFns ...func(*Options)) (*PutBucketPolicyOutput, error) {
	if params == nil {
		params = &PutBucketPolicyInput{}
	}

	result, metadata, err := c.invokeOperation(ctx, "PutBucketPolicy", params, optFns, c.addOperationPutBucketPolicyMiddlewares)
	if err != nil {
		return nil, err
	}

	out := result.(*PutBucketPolicyOutput)
	out.ResultMetadata = metadata
	return out, nil
}

type PutBucketPolicyInput struct {

	// The name of the bucket.
	//
	// Directory buckets - When you use this operation with a directory bucket, you
	// must use path-style requests in the format
	// https://s3express-control.region-code.amazonaws.com/bucket-name .
	// Virtual-hosted-style requests aren't supported. Directory bucket names must be
	// unique in the chosen Zone (Availability Zone or Local Zone). Bucket names must
	// also follow the format bucket-base-name--zone-id--x-s3 (for example,
	// DOC-EXAMPLE-BUCKET--usw2-az1--x-s3 ). For information about bucket naming
	// restrictions, see [Directory bucket naming rules]in the Amazon S3 User Guide
	//
	// [Directory bucket naming rules]: https://docs.aws.amazon.com/AmazonS3/latest/userguide/directory-bucket-naming-rules.html
	//
	// This member is required.
	Bucket *string

	// The bucket policy as a JSON document.
	//
	// For directory buckets, the only IAM action supported in the bucket policy is
	// s3express:CreateSession .
	//
	// This member is required.
	Policy *string

	// Indicates the algorithm used to create the checksum for the object when you use
	// the SDK. This header will not provide any additional functionality if you don't
	// use the SDK. When you send this header, there must be a corresponding
	// x-amz-checksum-algorithm or x-amz-trailer header sent. Otherwise, Amazon S3
	// fails the request with the HTTP status code 400 Bad Request .
	//
	// For the x-amz-checksum-algorithm  header, replace  algorithm  with the
	// supported algorithm from the following list:
	//
	//   - CRC32
	//
	//   - CRC32C
	//
	//   - SHA1
	//
	//   - SHA256
	//
	// For more information, see [Checking object integrity] in the Amazon S3 User Guide.
	//
	// If the individual checksum value you provide through x-amz-checksum-algorithm
	// doesn't match the checksum algorithm you set through
	// x-amz-sdk-checksum-algorithm , Amazon S3 ignores any provided ChecksumAlgorithm
	// parameter and uses the checksum algorithm that matches the provided value in
	// x-amz-checksum-algorithm .
	//
	// For directory buckets, when you use Amazon Web Services SDKs, CRC32 is the
	// default checksum algorithm that's used for performance.
	//
	// [Checking object integrity]: https://docs.aws.amazon.com/AmazonS3/latest/userguide/checking-object-integrity.html
	ChecksumAlgorithm types.ChecksumAlgorithm

	// Set this parameter to true to confirm that you want to remove your permissions
	// to change this bucket policy in the future.
	//
	// This functionality is not supported for directory buckets.
	ConfirmRemoveSelfBucketAccess *bool

	// The MD5 hash of the request body.
	//
	// For requests made using the Amazon Web Services Command Line Interface (CLI) or
	// Amazon Web Services SDKs, this field is calculated automatically.
	//
	// This functionality is not supported for directory buckets.
	ContentMD5 *string

	// The account ID of the expected bucket owner. If the account ID that you provide
	// does not match the actual owner of the bucket, the request fails with the HTTP
	// status code 403 Forbidden (access denied).
	//
	// For directory buckets, this header is not supported in this API operation. If
	// you specify this header, the request fails with the HTTP status code 501 Not
	// Implemented .
	ExpectedBucketOwner *string

	noSmithyDocumentSerde
}

func (in *PutBucketPolicyInput) bindEndpointParams(p *EndpointParameters) {

	p.Bucket = in.Bucket
	p.UseS3ExpressControlEndpoint = ptr.Bool(true)
}

type PutBucketPolicyOutput struct {
	// Metadata pertaining to the operation's result.
	ResultMetadata middleware.Metadata

	noSmithyDocumentSerde
}

func (c *Client) addOperationPutBucketPolicyMiddlewares(stack *middleware.Stack, options Options) (err error) {
	if err := stack.Serialize.Add(&setOperationInputMiddleware{}, middleware.After); err != nil {
		return err
	}
	err = stack.Serialize.Add(&awsRestxml_serializeOpPutBucketPolicy{}, middleware.After)
	if err != nil {
		return err
	}
	err = stack.Deserialize.Add(&awsRestxml_deserializeOpPutBucketPolicy{}, middleware.After)
	if err != nil {
		return err
	}
	if err := addProtocolFinalizerMiddlewares(stack, options, "PutBucketPolicy"); err != nil {
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
	if err = addOpPutBucketPolicyValidationMiddleware(stack); err != nil {
		return err
	}
	if err = stack.Initialize.Add(newServiceMetadataMiddleware_opPutBucketPolicy(options.Region), middleware.Before); err != nil {
		return err
	}
	if err = addMetadataRetrieverMiddleware(stack); err != nil {
		return err
	}
	if err = addRecursionDetection(stack); err != nil {
		return err
	}
	if err = addPutBucketPolicyInputChecksumMiddlewares(stack, options); err != nil {
		return err
	}
	if err = addPutBucketPolicyUpdateEndpoint(stack, options); err != nil {
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
	if err = s3cust.AddExpressDefaultChecksumMiddleware(stack); err != nil {
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

func (v *PutBucketPolicyInput) bucket() (string, bool) {
	if v.Bucket == nil {
		return "", false
	}
	return *v.Bucket, true
}

func newServiceMetadataMiddleware_opPutBucketPolicy(region string) *awsmiddleware.RegisterServiceMetadata {
	return &awsmiddleware.RegisterServiceMetadata{
		Region:        region,
		ServiceID:     ServiceID,
		OperationName: "PutBucketPolicy",
	}
}

// getPutBucketPolicyRequestAlgorithmMember gets the request checksum algorithm
// value provided as input.
func getPutBucketPolicyRequestAlgorithmMember(input interface{}) (string, bool) {
	in := input.(*PutBucketPolicyInput)
	if len(in.ChecksumAlgorithm) == 0 {
		return "", false
	}
	return string(in.ChecksumAlgorithm), true
}

func addPutBucketPolicyInputChecksumMiddlewares(stack *middleware.Stack, options Options) error {
	return internalChecksum.AddInputMiddleware(stack, internalChecksum.InputMiddlewareOptions{
		GetAlgorithm:                     getPutBucketPolicyRequestAlgorithmMember,
		RequireChecksum:                  true,
		EnableTrailingChecksum:           false,
		EnableComputeSHA256PayloadHash:   true,
		EnableDecodedContentLengthHeader: true,
	})
}

// getPutBucketPolicyBucketMember returns a pointer to string denoting a provided
// bucket member valueand a boolean indicating if the input has a modeled bucket
// name,
func getPutBucketPolicyBucketMember(input interface{}) (*string, bool) {
	in := input.(*PutBucketPolicyInput)
	if in.Bucket == nil {
		return nil, false
	}
	return in.Bucket, true
}
func addPutBucketPolicyUpdateEndpoint(stack *middleware.Stack, options Options) error {
	return s3cust.UpdateEndpoint(stack, s3cust.UpdateEndpointOptions{
		Accessor: s3cust.UpdateEndpointParameterAccessor{
			GetBucketFromInput: getPutBucketPolicyBucketMember,
		},
		UsePathStyle:                   options.UsePathStyle,
		UseAccelerate:                  options.UseAccelerate,
		SupportsAccelerate:             true,
		TargetS3ObjectLambda:           false,
		EndpointResolver:               options.EndpointResolver,
		EndpointResolverOptions:        options.EndpointOptions,
		UseARNRegion:                   options.UseARNRegion,
		DisableMultiRegionAccessPoints: options.DisableMultiRegionAccessPoints,
	})
}
