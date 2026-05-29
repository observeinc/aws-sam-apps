package metricsconfigurator

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
)

func TestS3URIToHTTPS(t *testing.T) {
	tests := []struct {
		name    string
		uri     string
		wantURL string
		wantErr bool
	}{
		{
			name:    "valid URI",
			uri:     "s3://observeinc/cloudwatchmetrics/filters/recommended.yaml",
			wantURL: "https://observeinc.s3.amazonaws.com/cloudwatchmetrics/filters/recommended.yaml",
		},
		{
			name:    "valid URI with simple key",
			uri:     "s3://mybucket/mykey.yaml",
			wantURL: "https://mybucket.s3.amazonaws.com/mykey.yaml",
		},
		{
			name:    "missing s3 prefix",
			uri:     "https://bucket/key",
			wantErr: true,
		},
		{
			name:    "no key",
			uri:     "s3://bucket",
			wantErr: true,
		},
		{
			name:    "empty key",
			uri:     "s3://bucket/",
			wantErr: true,
		},
		{
			name:    "empty string",
			uri:     "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url, err := s3URIToHTTPS(tt.uri)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if url != tt.wantURL {
				t.Errorf("url = %q, want %q", url, tt.wantURL)
			}
		})
	}
}

func TestParseFilterYAML(t *testing.T) {
	tests := []struct {
		name           string
		yaml           string
		wantIncludes   int
		wantExcludes   int
		wantErr        bool
		checkNamespace string
	}{
		{
			name: "empty filter (IncludeFilters with None namespace)",
			yaml: `IncludeFilters:
  - Namespace: "None"
`,
			wantIncludes:   1,
			wantExcludes:   0,
			checkNamespace: "None",
		},
		{
			name: "default filter",
			yaml: `IncludeFilters:
  - Namespace: "Default"
`,
			wantIncludes:   1,
			wantExcludes:   0,
			checkNamespace: "Default",
		},
		{
			name:         "full filter (empty ExcludeFilters)",
			yaml:         `ExcludeFilters: []`,
			wantIncludes: 0,
			wantExcludes: 0,
			wantErr:      true,
		},
		{
			name: "recommended filter with ExcludeFilters",
			yaml: `ExcludeFilters:
  - Namespace: AWS/RDS
    MetricNames:
      - AbortedClients
      - AuroraDMLRejectedMasterFull
  - Namespace: AWS/EC2
    MetricNames:
      - StatusCheckFailed
`,
			wantIncludes: 0,
			wantExcludes: 2,
		},
		{
			name: "include filters with metric names",
			yaml: `IncludeFilters:
  - Namespace: AWS/EC2
    MetricNames:
      - CPUUtilization
      - NetworkIn
  - Namespace: AWS/EBS
    MetricNames:
      - VolumeReadOps
`,
			wantIncludes: 2,
			wantExcludes: 0,
		},
		{
			name: "include filters without metric names",
			yaml: `IncludeFilters:
  - Namespace: AWS/EC2
  - Namespace: AWS/EBS
`,
			wantIncludes: 2,
			wantExcludes: 0,
		},
		{
			name:    "invalid yaml",
			yaml:    `{invalid: yaml: [`,
			wantErr: true,
		},
		{
			name:    "empty yaml",
			yaml:    ``,
			wantErr: true,
		},
		{
			name:    "yaml with neither field",
			yaml:    `SomethingElse: value`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseFilterYAML([]byte(tt.yaml))
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if len(result.IncludeFilters) != tt.wantIncludes {
				t.Errorf("IncludeFilters count = %d, want %d", len(result.IncludeFilters), tt.wantIncludes)
			}
			if len(result.ExcludeFilters) != tt.wantExcludes {
				t.Errorf("ExcludeFilters count = %d, want %d", len(result.ExcludeFilters), tt.wantExcludes)
			}
			if tt.checkNamespace != "" && len(result.IncludeFilters) > 0 {
				if *result.IncludeFilters[0].Namespace != tt.checkNamespace {
					t.Errorf("first IncludeFilter namespace = %q, want %q", *result.IncludeFilters[0].Namespace, tt.checkNamespace)
				}
			}
		})
	}
}

func TestParseFilterYAMLMetricNames(t *testing.T) {
	yaml := `ExcludeFilters:
  - Namespace: AWS/RDS
    MetricNames:
      - AbortedClients
      - AuroraDMLRejectedMasterFull
      - AuroraDMLRejectedWriterFull
`
	result, err := parseFilterYAML([]byte(yaml))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.ExcludeFilters) != 1 {
		t.Fatalf("expected 1 exclude filter, got %d", len(result.ExcludeFilters))
	}
	filter := result.ExcludeFilters[0]
	if *filter.Namespace != "AWS/RDS" {
		t.Errorf("namespace = %q, want AWS/RDS", *filter.Namespace)
	}
	expectedNames := []string{"AbortedClients", "AuroraDMLRejectedMasterFull", "AuroraDMLRejectedWriterFull"}
	if len(filter.MetricNames) != len(expectedNames) {
		t.Fatalf("metric names count = %d, want %d", len(filter.MetricNames), len(expectedNames))
	}
	for i, name := range filter.MetricNames {
		if name != expectedNames[i] {
			t.Errorf("metric name[%d] = %q, want %q", i, name, expectedNames[i])
		}
	}
}

func TestParseFilterYAMLPreservesTypes(t *testing.T) {
	yaml := `IncludeFilters:
  - Namespace: AWS/EC2
    MetricNames:
      - CPUUtilization
`
	result, err := parseFilterYAML([]byte(yaml))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	filter := result.IncludeFilters[0]
	_ = types.MetricStreamFilter{
		Namespace:   filter.Namespace,
		MetricNames: filter.MetricNames,
	}
}
