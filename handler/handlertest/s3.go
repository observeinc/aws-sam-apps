package handlertest

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type S3Client struct {
	CopyObjectFunc func(context.Context, *s3.CopyObjectInput, ...func(*s3.Options)) (*s3.CopyObjectOutput, error)
	PutObjectFunc  func(context.Context, *s3.PutObjectInput, ...func(*s3.Options)) (*s3.PutObjectOutput, error)
}

func (c *S3Client) CopyObject(ctx context.Context, params *s3.CopyObjectInput, optFns ...func(*s3.Options)) (*s3.CopyObjectOutput, error) {
	if c.CopyObjectFunc == nil {
		return nil, nil
	}
	return c.CopyObjectFunc(ctx, params, optFns...)
}

func (c *S3Client) PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	if c.PutObjectFunc == nil {
		return nil, nil
	}
	return c.PutObjectFunc(ctx, params, optFns...)
}
