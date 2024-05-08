package s3http

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/go-logr/logr"

	"github.com/observeinc/aws-sam-apps/handler/forwarder/s3http/internal/batch"
	"github.com/observeinc/aws-sam-apps/handler/forwarder/s3http/internal/decoders"
	"github.com/observeinc/aws-sam-apps/handler/forwarder/s3http/internal/request"
)

var ErrGetRequest = errors.New("failed to construct get request")

type GetObjectAPIClient interface {
	GetObject(context.Context, *s3.GetObjectInput, ...func(*s3.Options)) (*s3.GetObjectOutput, error)
}

// Client implements an S3 compatible shim layer that uploads data to HTTP.
type Client struct {
	GetObjectAPIClient
	RequestBuilder *request.Builder
}

func toGetInput(copyInput *s3.CopyObjectInput) *s3.GetObjectInput {
	parts := strings.SplitN(aws.ToString(copyInput.CopySource), "/", 2)
	if len(parts) != 2 {
		return nil
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
	}
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
		in.ContentType = ce
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

	getInput := toGetInput(params)
	if getInput == nil {
		return nil, ErrGetRequest
	}

	getResp, err := c.GetObjectAPIClient.GetObject(ctx, getInput, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to get object: %w", err)
	}
	defer getResp.Body.Close()
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

	dec, err := decoders.Get(aws.ToString(params.ContentEncoding), aws.ToString(params.ContentType))
	if err != nil {
		return nil, fmt.Errorf("failed to get decoder: %w", err)
	}

	err = batch.Run(ctx, &batch.RunInput{
		Decoder: dec(params.Body),
		Handler: c.RequestBuilder.With(map[string]string{
			"content-type": aws.ToString(params.ContentType),
			"key":          aws.ToString(params.Key),
		}),
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
		RequestBuilder: &request.Builder{
			URL:    cfg.DestinationURI,
			Client: cfg.HTTPClient,
		},
	}, nil
}
