package handler

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// ParseS3URI splits an s3://bucket/key URI into its bucket and key parts.
func ParseS3URI(uri string) (bucket, key string, err error) {
	if !strings.HasPrefix(uri, "s3://") {
		return "", "", fmt.Errorf("invalid S3 URI: must start with s3://")
	}
	path := strings.TrimPrefix(uri, "s3://")
	parts := strings.SplitN(path, "/", 2)
	if len(parts) < 2 || parts[1] == "" {
		return "", "", fmt.Errorf("invalid S3 URI: must contain bucket and key")
	}
	return parts[0], parts[1], nil
}

// GetS3Object fetches an s3://bucket/key object and returns the body as bytes.
// The caller interprets the bytes (YAML, JSON, etc.).
//
// Handles cross-region buckets: discovers the bucket's region via the
// x-amz-bucket-region header (manager.GetBucketRegion), then issues the
// GetObject from a region-specific client. Without this, an S3 client
// configured for the Lambda's region would 301 PermanentRedirect against a
// bucket living elsewhere.
func GetS3Object(ctx context.Context, awsCfg aws.Config, uri string) ([]byte, error) {
	bucket, key, err := ParseS3URI(uri)
	if err != nil {
		return nil, err
	}
	client := s3.NewFromConfig(awsCfg)

	bucketRegion, err := manager.GetBucketRegion(ctx, client, bucket)
	if err != nil {
		return nil, fmt.Errorf("failed to determine region for bucket %s: %w", bucket, err)
	}
	if bucketRegion != awsCfg.Region {
		client = s3.NewFromConfig(awsCfg, func(o *s3.Options) {
			o.Region = bucketRegion
		})
	}

	out, err := client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: &bucket,
		Key:    &key,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get s3://%s/%s: %w", bucket, key, err)
	}
	defer func() { _ = out.Body.Close() }()

	data, err := io.ReadAll(out.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read s3://%s/%s body: %w", bucket, key, err)
	}
	return data, nil
}
