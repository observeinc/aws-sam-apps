package metricsconfigurator

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
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

// s3URIToHTTPS converts s3://bucket/key to the virtual-hosted-style HTTPS URL.
func s3URIToHTTPS(uri string) (string, error) {
	if !strings.HasPrefix(uri, "s3://") {
		return "", fmt.Errorf("invalid S3 URI: must start with s3://")
	}
	path := strings.TrimPrefix(uri, "s3://")
	parts := strings.SplitN(path, "/", 2)
	if len(parts) < 2 || parts[1] == "" {
		return "", fmt.Errorf("invalid S3 URI: must contain bucket and key")
	}
	return fmt.Sprintf("https://%s.s3.amazonaws.com/%s", parts[0], parts[1]), nil
}

func downloadFilterYAML(ctx context.Context, filterURI string) ([]byte, error) {
	url, err := s3URIToHTTPS(filterURI)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request for %s: %w", url, err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to download filter YAML from %s: %w", url, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download filter YAML from %s: HTTP %d", url, resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read filter YAML body: %w", err)
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

	if len(result.IncludeFilters) == 0 && len(result.ExcludeFilters) == 0 {
		return nil, fmt.Errorf("filter YAML must contain IncludeFilters or ExcludeFilters")
	}

	return result, nil
}
