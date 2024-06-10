package awstest

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type S3Client struct {
	GetObjectFunc  func(context.Context, *s3.GetObjectInput, ...func(*s3.Options)) (*s3.GetObjectOutput, error)
	CopyObjectFunc func(context.Context, *s3.CopyObjectInput, ...func(*s3.Options)) (*s3.CopyObjectOutput, error)
	PutObjectFunc  func(context.Context, *s3.PutObjectInput, ...func(*s3.Options)) (*s3.PutObjectOutput, error)
	HeadBucketFunc func(context.Context, *s3.HeadBucketInput, ...func(*s3.Options)) (*s3.HeadBucketOutput, error)
}

func (c *S3Client) GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	if c.GetObjectFunc == nil {
		return nil, nil
	}
	return c.GetObjectFunc(ctx, params, optFns...)
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

func (c *S3Client) HeadBucket(ctx context.Context, params *s3.HeadBucketInput, optFns ...func(*s3.Options)) (*s3.HeadBucketOutput, error) {
	if c.HeadBucketFunc == nil {
		return nil, nil
	}
	return c.HeadBucketFunc(ctx, params, optFns...)
}

// FileGetter is a fake S3 client that grabs files from disk
type FileGetter struct {
	S3Client
	ContentType     *string
	ContentEncoding *string
}

func (c *FileGetter) GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	file, err := os.Open(fmt.Sprintf("%s/%s", aws.ToString(params.Bucket), aws.ToString(params.Key)))
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	fileinfo, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to read fileinfo: %w", err)
	}
	return &s3.GetObjectOutput{
		Body:            file,
		ContentLength:   aws.Int64(fileinfo.Size()),
		ContentType:     c.ContentType,
		ContentEncoding: c.ContentEncoding,
	}, nil
}
