package pollerconfigurator

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type TagFilter struct {
	Key    string   `json:"key"`
	Values []string `json:"values,omitempty"`
}

type ResourceFilter struct {
	ResourceType  string      `json:"resourceType,omitempty"`
	Pattern       string      `json:"pattern,omitempty"`
	DimensionName string      `json:"dimensionName,omitempty"`
	TagFilters    []TagFilter `json:"tagFilters"`
}

type DimensionFilter struct {
	Name  string `json:"name"`
	Value string `json:"value,omitempty"`
}

type QueryConfig struct {
	Namespace      string            `json:"namespace"`
	MetricNames    []string          `json:"metricNames,omitempty"`
	Dimensions     []DimensionFilter `json:"dimensions,omitempty"`
	ResourceFilter *ResourceFilter   `json:"resourceFilter,omitempty"`
}

type PollerConfig struct {
	Name         string        `json:"name"`
	DatastreamId string        `json:"datastreamId"`
	Period       int64         `json:"period"`
	Delay        int64         `json:"delay"`
	Interval     string        `json:"interval"`
	Retries      *int64        `json:"retries,omitempty"`
	Queries      []QueryConfig `json:"queries"`
}

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

func downloadPollerConfig(ctx context.Context, uri string, awsCfg aws.Config) (*PollerConfig, error) {
	// for testing: allow direct HTTP URLs so tests can use httptest servers
	if strings.HasPrefix(uri, "http://") || strings.HasPrefix(uri, "https://") {
		return downloadPollerConfigFromURL(ctx, uri)
	}

	bucket, key, err := parseS3URI(uri)
	if err != nil {
		return nil, err
	}
	return downloadPollerConfigFromS3(ctx, awsCfg, bucket, key)
}

func downloadPollerConfigFromS3(ctx context.Context, awsCfg aws.Config, bucket, key string) (*PollerConfig, error) {
	client := s3.NewFromConfig(awsCfg)
	out, err := client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: &bucket,
		Key:    &key,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get s3://%s/%s: %w", bucket, key, err)
	}
	defer func() { _ = out.Body.Close() }()

	return parsePollerConfig(out.Body)
}

func downloadPollerConfigFromURL(ctx context.Context, url string) (*PollerConfig, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request for %s: %w", url, err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to download poller config from %s: %w", url, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download poller config from %s: HTTP %d", url, resp.StatusCode)
	}

	return parsePollerConfig(resp.Body)
}

func parsePollerConfig(r io.Reader) (*PollerConfig, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read poller config body: %w", err)
	}

	var cfg PollerConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse poller config JSON: %w", err)
	}

	if len(cfg.Queries) == 0 {
		return nil, fmt.Errorf("poller config must contain at least one query")
	}
	if cfg.Period <= 0 {
		return nil, fmt.Errorf("poller config period must be positive")
	}
	if cfg.Delay <= 0 {
		return nil, fmt.Errorf("poller config delay must be positive")
	}
	if cfg.DatastreamId == "" {
		return nil, fmt.Errorf("poller config must include datastreamId")
	}
	if cfg.Interval == "" {
		return nil, fmt.Errorf("poller config must include interval")
	}

	return &cfg, nil
}
