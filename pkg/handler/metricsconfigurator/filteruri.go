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
