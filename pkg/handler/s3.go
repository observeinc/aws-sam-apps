package handler

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
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
func GetS3Object(ctx context.Context, awsCfg aws.Config, uri string) ([]byte, error) {
	bucket, key, err := ParseS3URI(uri)
	if err != nil {
		return nil, err
	}
	client := s3.NewFromConfig(awsCfg)
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
