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
	"time"
)

// Retrieves all the metadata from an object without returning the object itself.
// This operation is useful if you're interested only in an object's metadata.
// GetObjectAttributes combines the functionality of HeadObject and ListParts . All
// of the data returned with each of those individual calls can be returned with a
// single call to GetObjectAttributes . Directory buckets - For directory buckets,
// you must make requests for this API operation to the Zonal endpoint. These
// endpoints support virtual-hosted-style requests in the format
// https://bucket_name.s3express-az_id.region.amazonaws.com/key-name . Path-style
// requests are not supported. For more information, see Regional and Zonal
// endpoints (https://docs.aws.amazon.com/AmazonS3/latest/userguide/s3-express-Regions-and-Zones.html)
// in the Amazon S3 User Guide. Permissions
//   - General purpose bucket permissions - To use GetObjectAttributes , you must
//     have READ access to the object. The permissions that you need to use this
//     operation with depend on whether the bucket is versioned. If the bucket is
//     versioned, you need both the s3:GetObjectVersion and
//     s3:GetObjectVersionAttributes permissions for this operation. If the bucket is
//     not versioned, you need the s3:GetObject and s3:GetObjectAttributes
//     permissions. For more information, see Specifying Permissions in a Policy (https://docs.aws.amazon.com/AmazonS3/latest/dev/using-with-s3-actions.html)
//     in the Amazon S3 User Guide. If the object that you request does not exist, the
//     error Amazon S3 returns depends on whether you also have the s3:ListBucket
//     permission.
//   - If you have the s3:ListBucket permission on the bucket, Amazon S3 returns an
//     HTTP status code 404 Not Found ("no such key") error.
//   - If you don't have the s3:ListBucket permission, Amazon S3 returns an HTTP
//     status code 403 Forbidden ("access denied") error.
//   - Directory bucket permissions - To grant access to this API operation on a
//     directory bucket, we recommend that you use the CreateSession (https://docs.aws.amazon.com/AmazonS3/latest/API/API_CreateSession.html)
//     API operation for session-based authorization. Specifically, you grant the
//     s3express:CreateSession permission to the directory bucket in a bucket policy
//     or an IAM identity-based policy. Then, you make the CreateSession API call on
//     the bucket to obtain a session token. With the session token in your request
//     header, you can make API requests to this operation. After the session token
//     expires, you make another CreateSession API call to generate a new session
//     token for use. Amazon Web Services CLI or SDKs create session and refresh the
//     session token automatically to avoid service interruptions when a session
//     expires. For more information about authorization, see CreateSession (https://docs.aws.amazon.com/AmazonS3/latest/API/API_CreateSession.html)
//     .
//
// Encryption Encryption request headers, like x-amz-server-side-encryption ,
// should not be sent for HEAD requests if your object uses server-side encryption
// with Key Management Service (KMS) keys (SSE-KMS), dual-layer server-side
// encryption with Amazon Web Services KMS keys (DSSE-KMS), or server-side
// encryption with Amazon S3 managed encryption keys (SSE-S3). The
// x-amz-server-side-encryption header is used when you PUT an object to S3 and
// want to specify the encryption method. If you include this header in a GET
// request for an object that uses these types of keys, you’ll get an HTTP 400 Bad
// Request error. It's because the encryption method can't be changed when you
// retrieve the object. If you encrypt an object by using server-side encryption
// with customer-provided encryption keys (SSE-C) when you store the object in
// Amazon S3, then when you retrieve the metadata from the object, you must use the
// following headers to provide the encryption key for the server to be able to
// retrieve the object's metadata. The headers are:
//   - x-amz-server-side-encryption-customer-algorithm
//   - x-amz-server-side-encryption-customer-key
//   - x-amz-server-side-encryption-customer-key-MD5
//
// For more information about SSE-C, see Server-Side Encryption (Using
// Customer-Provided Encryption Keys) (https://docs.aws.amazon.com/AmazonS3/latest/dev/ServerSideEncryptionCustomerKeys.html)
// in the Amazon S3 User Guide. Directory bucket permissions - For directory
// buckets, only server-side encryption with Amazon S3 managed keys (SSE-S3) (
// AES256 ) is supported. Versioning Directory buckets - S3 Versioning isn't
// enabled and supported for directory buckets. For this API operation, only the
// null value of the version ID is supported by directory buckets. You can only
// specify null to the versionId query parameter in the request. Conditional
// request headers Consider the following when using request headers:
//   - If both of the If-Match and If-Unmodified-Since headers are present in the
//     request as follows, then Amazon S3 returns the HTTP status code 200 OK and the
//     data requested:
//   - If-Match condition evaluates to true .
//   - If-Unmodified-Since condition evaluates to false . For more information
//     about conditional requests, see RFC 7232 (https://tools.ietf.org/html/rfc7232)
//     .
//   - If both of the If-None-Match and If-Modified-Since headers are present in
//     the request as follows, then Amazon S3 returns the HTTP status code 304 Not
//     Modified :
//   - If-None-Match condition evaluates to false .
//   - If-Modified-Since condition evaluates to true . For more information about
//     conditional requests, see RFC 7232 (https://tools.ietf.org/html/rfc7232) .
//
// HTTP Host header syntax Directory buckets - The HTTP Host header syntax is
// Bucket_name.s3express-az_id.region.amazonaws.com . The following actions are
// related to GetObjectAttributes :
//   - GetObject (https://docs.aws.amazon.com/AmazonS3/latest/API/API_GetObject.html)
//   - GetObjectAcl (https://docs.aws.amazon.com/AmazonS3/latest/API/API_GetObjectAcl.html)
//   - GetObjectLegalHold (https://docs.aws.amazon.com/AmazonS3/latest/API/API_GetObjectLegalHold.html)
//   - GetObjectLockConfiguration (https://docs.aws.amazon.com/AmazonS3/latest/API/API_GetObjectLockConfiguration.html)
//   - GetObjectRetention (https://docs.aws.amazon.com/AmazonS3/latest/API/API_GetObjectRetention.html)
//   - GetObjectTagging (https://docs.aws.amazon.com/AmazonS3/latest/API/API_GetObjectTagging.html)
//   - HeadObject (https://docs.aws.amazon.com/AmazonS3/latest/API/API_HeadObject.html)
//   - ListParts (https://docs.aws.amazon.com/AmazonS3/latest/API/API_ListParts.html)
func (c *Client) GetObjectAttributes(ctx context.Context, params *GetObjectAttributesInput, optFns ...func(*Options)) (*GetObjectAttributesOutput, error) {
	if params == nil {
		params = &GetObjectAttributesInput{}
	}

	result, metadata, err := c.invokeOperation(ctx, "GetObjectAttributes", params, optFns, c.addOperationGetObjectAttributesMiddlewares)
	if err != nil {
		return nil, err
	}

	out := result.(*GetObjectAttributesOutput)
	out.ResultMetadata = metadata
	return out, nil
}

type GetObjectAttributesInput struct {

	// The name of the bucket that contains the object. Directory buckets - When you
	// use this operation with a directory bucket, you must use virtual-hosted-style
	// requests in the format Bucket_name.s3express-az_id.region.amazonaws.com .
	// Path-style requests are not supported. Directory bucket names must be unique in
	// the chosen Availability Zone. Bucket names must follow the format
	// bucket_base_name--az-id--x-s3 (for example,  DOC-EXAMPLE-BUCKET--usw2-az2--x-s3
	// ). For information about bucket naming restrictions, see Directory bucket
	// naming rules (https://docs.aws.amazon.com/AmazonS3/latest/userguide/directory-bucket-naming-rules.html)
	// in the Amazon S3 User Guide. Access points - When you use this action with an
	// access point, you must provide the alias of the access point in place of the
	// bucket name or specify the access point ARN. When using the access point ARN,
	// you must direct requests to the access point hostname. The access point hostname
	// takes the form AccessPointName-AccountId.s3-accesspoint.Region.amazonaws.com.
	// When using this action with an access point through the Amazon Web Services
	// SDKs, you provide the access point ARN in place of the bucket name. For more
	// information about access point ARNs, see Using access points (https://docs.aws.amazon.com/AmazonS3/latest/userguide/using-access-points.html)
	// in the Amazon S3 User Guide. Access points and Object Lambda access points are
	// not supported by directory buckets. S3 on Outposts - When you use this action
	// with Amazon S3 on Outposts, you must direct requests to the S3 on Outposts
	// hostname. The S3 on Outposts hostname takes the form
	// AccessPointName-AccountId.outpostID.s3-outposts.Region.amazonaws.com . When you
	// use this action with S3 on Outposts through the Amazon Web Services SDKs, you
	// provide the Outposts access point ARN in place of the bucket name. For more
	// information about S3 on Outposts ARNs, see What is S3 on Outposts? (https://docs.aws.amazon.com/AmazonS3/latest/userguide/S3onOutposts.html)
	// in the Amazon S3 User Guide.
	//
	// This member is required.
	Bucket *string

	// The object key.
	//
	// This member is required.
	Key *string

	// Specifies the fields at the root level that you want returned in the response.
	// Fields that you do not specify are not returned.
	//
	// This member is required.
	ObjectAttributes []types.ObjectAttributes

	// The account ID of the expected bucket owner. If the account ID that you provide
	// does not match the actual owner of the bucket, the request fails with the HTTP
	// status code 403 Forbidden (access denied).
	ExpectedBucketOwner *string

	// Sets the maximum number of parts to return.
	MaxParts *int32

	// Specifies the part after which listing should begin. Only parts with higher
	// part numbers will be listed.
	PartNumberMarker *string

	// Confirms that the requester knows that they will be charged for the request.
	// Bucket owners need not specify this parameter in their requests. If either the
	// source or destination S3 bucket has Requester Pays enabled, the requester will
	// pay for corresponding charges to copy the object. For information about
	// downloading objects from Requester Pays buckets, see Downloading Objects in
	// Requester Pays Buckets (https://docs.aws.amazon.com/AmazonS3/latest/dev/ObjectsinRequesterPaysBuckets.html)
	// in the Amazon S3 User Guide. This functionality is not supported for directory
	// buckets.
	RequestPayer types.RequestPayer

	// Specifies the algorithm to use when encrypting the object (for example,
	// AES256). This functionality is not supported for directory buckets.
	SSECustomerAlgorithm *string

	// Specifies the customer-provided encryption key for Amazon S3 to use in
	// encrypting data. This value is used to store the object and then it is
	// discarded; Amazon S3 does not store the encryption key. The key must be
	// appropriate for use with the algorithm specified in the
	// x-amz-server-side-encryption-customer-algorithm header. This functionality is
	// not supported for directory buckets.
	SSECustomerKey *string

	// Specifies the 128-bit MD5 digest of the encryption key according to RFC 1321.
	// Amazon S3 uses this header for a message integrity check to ensure that the
	// encryption key was transmitted without error. This functionality is not
	// supported for directory buckets.
	SSECustomerKeyMD5 *string

	// The version ID used to reference a specific version of the object. S3
	// Versioning isn't enabled and supported for directory buckets. For this API
	// operation, only the null value of the version ID is supported by directory
	// buckets. You can only specify null to the versionId query parameter in the
	// request.
	VersionId *string

	noSmithyDocumentSerde
}

func (in *GetObjectAttributesInput) bindEndpointParams(p *EndpointParameters) {
	p.Bucket = in.Bucket

}

type GetObjectAttributesOutput struct {

	// The checksum or digest of the object.
	Checksum *types.Checksum

	// Specifies whether the object retrieved was ( true ) or was not ( false ) a
	// delete marker. If false , this response header does not appear in the response.
	// This functionality is not supported for directory buckets.
	DeleteMarker *bool

	// An ETag is an opaque identifier assigned by a web server to a specific version
	// of a resource found at a URL.
	ETag *string

	// The creation date of the object.
	LastModified *time.Time

	// A collection of parts associated with a multipart upload.
	ObjectParts *types.GetObjectAttributesParts

	// The size of the object in bytes.
	ObjectSize *int64

	// If present, indicates that the requester was successfully charged for the
	// request. This functionality is not supported for directory buckets.
	RequestCharged types.RequestCharged

	// Provides the storage class information of the object. Amazon S3 returns this
	// header for all objects except for S3 Standard storage class objects. For more
	// information, see Storage Classes (https://docs.aws.amazon.com/AmazonS3/latest/dev/storage-class-intro.html)
	// . Directory buckets - Only the S3 Express One Zone storage class is supported by
	// directory buckets to store objects.
	StorageClass types.StorageClass

	// The version ID of the object. This functionality is not supported for directory
	// buckets.
	VersionId *string

	// Metadata pertaining to the operation's result.
	ResultMetadata middleware.Metadata

	noSmithyDocumentSerde
}

func (c *Client) addOperationGetObjectAttributesMiddlewares(stack *middleware.Stack, options Options) (err error) {
	if err := stack.Serialize.Add(&setOperationInputMiddleware{}, middleware.After); err != nil {
		return err
	}
	err = stack.Serialize.Add(&awsRestxml_serializeOpGetObjectAttributes{}, middleware.After)
	if err != nil {
		return err
	}
	err = stack.Deserialize.Add(&awsRestxml_deserializeOpGetObjectAttributes{}, middleware.After)
	if err != nil {
		return err
	}
	if err := addProtocolFinalizerMiddlewares(stack, options, "GetObjectAttributes"); err != nil {
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
	if err = addPutBucketContextMiddleware(stack); err != nil {
		return err
	}
	if err = addOpGetObjectAttributesValidationMiddleware(stack); err != nil {
		return err
	}
	if err = stack.Initialize.Add(newServiceMetadataMiddleware_opGetObjectAttributes(options.Region), middleware.Before); err != nil {
		return err
	}
	if err = addMetadataRetrieverMiddleware(stack); err != nil {
		return err
	}
	if err = awsmiddleware.AddRecursionDetection(stack); err != nil {
		return err
	}
	if err = addGetObjectAttributesUpdateEndpoint(stack, options); err != nil {
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

func (v *GetObjectAttributesInput) bucket() (string, bool) {
	if v.Bucket == nil {
		return "", false
	}
	return *v.Bucket, true
}

func newServiceMetadataMiddleware_opGetObjectAttributes(region string) *awsmiddleware.RegisterServiceMetadata {
	return &awsmiddleware.RegisterServiceMetadata{
		Region:        region,
		ServiceID:     ServiceID,
		OperationName: "GetObjectAttributes",
	}
}

// getGetObjectAttributesBucketMember returns a pointer to string denoting a
// provided bucket member valueand a boolean indicating if the input has a modeled
// bucket name,
func getGetObjectAttributesBucketMember(input interface{}) (*string, bool) {
	in := input.(*GetObjectAttributesInput)
	if in.Bucket == nil {
		return nil, false
	}
	return in.Bucket, true
}
func addGetObjectAttributesUpdateEndpoint(stack *middleware.Stack, options Options) error {
	return s3cust.UpdateEndpoint(stack, s3cust.UpdateEndpointOptions{
		Accessor: s3cust.UpdateEndpointParameterAccessor{
			GetBucketFromInput: getGetObjectAttributesBucketMember,
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
