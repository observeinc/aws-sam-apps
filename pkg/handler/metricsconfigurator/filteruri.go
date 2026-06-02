package metricsconfigurator

import (
	"fmt"

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

type ParsedFilters struct {
	IncludeFilters []types.MetricStreamFilter
	ExcludeFilters []types.MetricStreamFilter
}

func parseFilterYAML(data []byte) (*ParsedFilters, error) {
	var raw FilterYAML
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse filter YAML: %w", err)
	}

	if len(raw.IncludeFilters) > 0 && len(raw.ExcludeFilters) > 0 {
		return nil, fmt.Errorf("filter YAML cannot specify both IncludeFilters and ExcludeFilters; CloudWatch's PutMetricStream allows only one")
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
