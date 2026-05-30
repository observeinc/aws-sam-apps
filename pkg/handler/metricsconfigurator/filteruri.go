package metricsconfigurator

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"gopkg.in/yaml.v3"
)

type FilterYAML struct {
	IncludeFilters []FilterEntry `yaml:"IncludeFilters,omitempty"`
	ExcludeFilters []FilterEntry `yaml:"ExcludeFilters,omitempty"`
}

type FilterEntry struct {
	Namespace   string   `yaml:"Namespace"`
	MetricNames []string `yaml:"MetricNames,omitempty"`
}

// parseS3URI splits an s3://bucket/key URI into its bucket and key parts.
func parseS3URI(uri string) (bucket, key string, err error) {
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

func downloadFilterYAML(ctx context.Context, cfg aws.Config, filterURI string) ([]byte, error) {
	bucket, key, err := parseS3URI(filterURI)
	if err != nil {
		return nil, err
	}

	s3Client := s3.NewFromConfig(cfg)
	out, err := s3Client.GetObject(ctx, &s3.GetObjectInput{
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

type ParsedFilters struct {
	IncludeFilters []types.MetricStreamFilter
	ExcludeFilters []types.MetricStreamFilter
}

func parseFilterYAML(data []byte) (*ParsedFilters, error) {
	var raw FilterYAML
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse filter YAML: %w", err)
	}

	// Distinguish "key present but empty list" (e.g. `ExcludeFilters: []` in
	// full.yaml, meaning "stream everything") from "key absent" (malformed
	// or unrelated content) by re-parsing as a map.
	var keys map[string]interface{}
	_ = yaml.Unmarshal(data, &keys)
	_, hasInclude := keys["IncludeFilters"]
	_, hasExclude := keys["ExcludeFilters"]
	if !hasInclude && !hasExclude {
		return nil, fmt.Errorf("filter YAML must contain IncludeFilters or ExcludeFilters")
	}

	result := &ParsedFilters{}

	for _, entry := range raw.IncludeFilters {
		ns := entry.Namespace
		result.IncludeFilters = append(result.IncludeFilters, types.MetricStreamFilter{
			Namespace:   &ns,
			MetricNames: entry.MetricNames,
		})
	}

	for _, entry := range raw.ExcludeFilters {
		ns := entry.Namespace
		result.ExcludeFilters = append(result.ExcludeFilters, types.MetricStreamFilter{
			Namespace:   &ns,
			MetricNames: entry.MetricNames,
		})
	}

	return result, nil
}
