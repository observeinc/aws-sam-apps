package s3http

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/go-logr/logr"

	"github.com/observeinc/aws-sam-apps/pkg/handler/forwarder/s3http/internal/batch"
	"github.com/observeinc/aws-sam-apps/pkg/handler/forwarder/s3http/internal/decoders"
	"github.com/observeinc/aws-sam-apps/pkg/handler/forwarder/s3http/internal/request"
)

var (
	errInvalidCopySource = fmt.Errorf("invalid copy source")
	errNoObjectInput     = fmt.Errorf("no object input")
	errMissingBucket     = fmt.Errorf("missing bucket")
	errMissingKey        = fmt.Errorf("missing key")
	errMissingBody       = fmt.Errorf("missing body")
)

type GetObjectAPIClient interface {
	GetObject(context.Context, *s3.GetObjectInput, ...func(*s3.Options)) (*s3.GetObjectOutput, error)
}

// Client implements an S3 compatible shim layer that uploads data to HTTP.
type Client struct {
	GetObjectAPIClient
	RequestBuilder *request.Builder
	GzipLevel      *int
}

func toGetInput(copyInput *s3.CopyObjectInput) (*s3.GetObjectInput, error) {
	parts := strings.SplitN(aws.ToString(copyInput.CopySource), "/", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("%w: %q", errInvalidCopySource, aws.ToString(copyInput.CopySource))
	}

	return &s3.GetObjectInput{
		Bucket:               aws.String(parts[0]),
		Key:                  aws.String(parts[1]),
		ExpectedBucketOwner:  copyInput.ExpectedSourceBucketOwner,
		IfMatch:              copyInput.CopySourceIfMatch,
		IfModifiedSince:      copyInput.CopySourceIfModifiedSince,
		IfNoneMatch:          copyInput.CopySourceIfNoneMatch,
		IfUnmodifiedSince:    copyInput.CopySourceIfUnmodifiedSince,
		RequestPayer:         copyInput.RequestPayer,
		SSECustomerAlgorithm: copyInput.CopySourceSSECustomerAlgorithm,
		SSECustomerKey:       copyInput.CopySourceSSECustomerKey,
		SSECustomerKeyMD5:    copyInput.CopySourceSSECustomerKeyMD5,
	}, nil
}

func toPutInput(copyInput *s3.CopyObjectInput, getOutput *s3.GetObjectOutput) *s3.PutObjectInput {
	in := &s3.PutObjectInput{
		Bucket:              copyInput.Bucket,
		Key:                 copyInput.Key,
		ACL:                 copyInput.ACL,
		Body:                getOutput.Body,
		BucketKeyEnabled:    copyInput.BucketKeyEnabled,
		ContentDisposition:  copyInput.ContentDisposition,
		ContentEncoding:     getOutput.ContentEncoding,
		ContentLanguage:     copyInput.ContentLanguage,
		ContentType:         getOutput.ContentType,
		ExpectedBucketOwner: copyInput.ExpectedBucketOwner,
		Expires:             copyInput.Expires,
		GrantFullControl:    copyInput.GrantFullControl,
		GrantReadACP:        copyInput.GrantReadACP,
		GrantWriteACP:       copyInput.GrantWriteACP,
		// TODO: respect MetadataDirective
		Metadata:                  copyInput.Metadata,
		ObjectLockLegalHoldStatus: copyInput.ObjectLockLegalHoldStatus,
		ObjectLockMode:            copyInput.ObjectLockMode,
		ObjectLockRetainUntilDate: copyInput.ObjectLockRetainUntilDate,
		RequestPayer:              copyInput.RequestPayer,
		SSECustomerAlgorithm:      copyInput.SSECustomerAlgorithm,
		SSECustomerKey:            copyInput.SSECustomerKey,
		SSECustomerKeyMD5:         copyInput.SSECustomerKeyMD5,
		SSEKMSEncryptionContext:   copyInput.SSEKMSEncryptionContext,
		SSEKMSKeyId:               copyInput.SSEKMSKeyId,
		ServerSideEncryption:      copyInput.ServerSideEncryption,
		StorageClass:              copyInput.StorageClass,
		Tagging:                   copyInput.Tagging,
		// TODO: respect TaggingDirective
		WebsiteRedirectLocation: copyInput.WebsiteRedirectLocation,
	}

	if ct := copyInput.ContentType; ct != nil {
		in.ContentType = ct
	}
	if ce := copyInput.ContentEncoding; ce != nil {
		in.ContentEncoding = ce
	}
	return in
}

func toCopyOutput(*s3.PutObjectOutput) *s3.CopyObjectOutput {
	return nil
}

// CopyObject is treated as a GetObject call with our S3 client, and a PutObject to our HTTP destination.
func (c *Client) CopyObject(ctx context.Context, params *s3.CopyObjectInput, opts ...func(*s3.Options)) (*s3.CopyObjectOutput, error) {
	logger := logr.FromContextOrDiscard(ctx)
	logger.V(6).Info("processing CopyObject", "copyObjectInput", params)

	getInput, err := toGetInput(params)
	if err != nil {
		return nil, fmt.Errorf("failed to copy object: %w", err)
	}

	getResp, err := c.GetObject(ctx, getInput, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to get object: %w", err)
	}
	defer func() {
		if closeErr := getResp.Body.Close(); closeErr != nil && err == nil {
			logger.Error(closeErr, "failed to close response body")
		}
	}()

	if getResp.ContentLength != nil && *getResp.ContentLength == 0 {
		logger.V(6).Info("skipping empty file")
		return toCopyOutput(nil), nil
	}

	putResp, err := c.PutObject(ctx, toPutInput(params, getResp), opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to put object: %w", err)
	}
	return toCopyOutput(putResp), nil
}

// PutObject uploads to HTTP destination.
func (c *Client) PutObject(ctx context.Context, params *s3.PutObjectInput, _ ...func(*s3.Options)) (out *s3.PutObjectOutput, err error) {
	logger := logr.FromContextOrDiscard(ctx)
	logger.V(6).Info("processing PutObject", "putObjectInput", params)

	switch {
	case params == nil:
		return nil, errNoObjectInput
	case aws.ToString(params.Bucket) == "":
		return nil, errMissingBucket
	case aws.ToString(params.Key) == "":
		return nil, errMissingKey
	case params.Body == nil:
		return nil, errMissingBody
	}

	dec, err := decoders.Get(aws.ToString(params.ContentEncoding), aws.ToString(params.ContentType), params.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to get decoder: %w", err)
	}

	headers := map[string]string{
		"Content-Type": "application/x-ndjson",
	}
	if c.GzipLevel != nil {
		headers["Content-Encoding"] = "gzip"
	}

	err = batch.Run(ctx, &batch.RunInput{
		Decoder:   dec,
		GzipLevel: c.GzipLevel,
		Handler: c.RequestBuilder.With(map[string]string{
			"content-type": aws.ToString(params.ContentType),
			"key":          aws.ToString(params.Key),
		}, headers),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to process: %w", err)
	}
	return
}

func New(cfg *Config) (*Client, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("failed to validate config: %w", err)
	}

	return &Client{
		GetObjectAPIClient: cfg.GetObjectAPIClient,
		GzipLevel:          cfg.GzipLevel,
		RequestBuilder: &request.Builder{
			URL:    cfg.DestinationURI,
			Client: cfg.HTTPClient,
		},
	}, nil
}
