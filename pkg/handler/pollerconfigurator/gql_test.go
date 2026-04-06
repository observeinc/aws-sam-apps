package pollerconfigurator

import (
	"strings"
	"testing"
)

func TestBuildQueriesGQL(t *testing.T) {
	tests := []struct {
		name     string
		queries  []QueryConfig
		contains []string
	}{
		{
			name: "simple namespace",
			queries: []QueryConfig{
				{Namespace: "AWS/EC2"},
			},
			contains: []string{`namespace: "AWS/EC2"`},
		},
		{
			name: "with metric names",
			queries: []QueryConfig{
				{
					Namespace:   "AWS/EC2",
					MetricNames: []string{"CPUUtilization", "NetworkIn"},
				},
			},
			contains: []string{
				`namespace: "AWS/EC2"`,
				`metricNames: ["CPUUtilization", "NetworkIn"]`,
			},
		},
		{
			name: "with dimensions",
			queries: []QueryConfig{
				{
					Namespace: "AWS/EC2",
					Dimensions: []DimensionFilter{
						{Name: "InstanceId", Value: "i-1234"},
					},
				},
			},
			contains: []string{
				`dimensions: [{name: "InstanceId", value: "i-1234"}]`,
			},
		},
		{
			name: "with resource filter and tag filters",
			queries: []QueryConfig{
				{
					Namespace: "AWS/EC2",
					ResourceFilter: &ResourceFilter{
						ResourceType: "AWS::EC2::Instance",
						TagFilters: []TagFilter{
							{Key: "Environment", Values: []string{"prod", "staging"}},
						},
					},
				},
			},
			contains: []string{
				`resourceType: "AWS::EC2::Instance"`,
				`key: "Environment"`,
				`values: ["prod", "staging"]`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildQueriesGQL(tt.queries)
			for _, substr := range tt.contains {
				if !strings.Contains(result, substr) {
					t.Errorf("buildQueriesGQL() = %q, missing %q", result, substr)
				}
			}
		})
	}
}

func TestBuildCreatePollerMutation(t *testing.T) {
	cfg := &PollerConfig{
		Name:         "test-poller",
		DatastreamId: "ds-999",
		Period:       300,
		Delay:        300,
		Queries: []QueryConfig{
			{Namespace: "AWS/EC2", MetricNames: []string{"CPUUtilization"}},
		},
	}

	result := buildCreatePollerMutation("12345", cfg, "us-east-1", "arn:aws:iam::999:role/observe")

	expects := []string{
		`mutation`,
		`createPoller`,
		`workspaceId: "12345"`,
		`name: "test-poller"`,
		`datastreamId: "ds-999"`,
		`period: "300"`,
		`delay: "300"`,
		`region: "us-east-1"`,
		`assumeRoleArn: "arn:aws:iam::999:role/observe"`,
		`namespace: "AWS/EC2"`,
		`metricNames: ["CPUUtilization"]`,
		`{ id name }`,
	}

	for _, exp := range expects {
		if !strings.Contains(result, exp) {
			t.Errorf("buildCreatePollerMutation() missing %q in:\n%s", exp, result)
		}
	}
}

func TestBuildUpdatePollerMutation(t *testing.T) {
	cfg := &PollerConfig{
		Name:         "updated-poller",
		DatastreamId: "ds-111",
		Period:       600,
		Delay:        600,
		Queries: []QueryConfig{
			{Namespace: "AWS/Lambda"},
		},
	}

	result := buildUpdatePollerMutation("67890", cfg, "eu-west-1", "arn:aws:iam::111:role/obs")

	expects := []string{
		`mutation`,
		`updatePoller`,
		`id: "67890"`,
		`name: "updated-poller"`,
		`datastreamId: "ds-111"`,
		`period: "600"`,
		`region: "eu-west-1"`,
	}

	for _, exp := range expects {
		if !strings.Contains(result, exp) {
			t.Errorf("buildUpdatePollerMutation() missing %q in:\n%s", exp, result)
		}
	}
}

func TestBuildDeletePollerMutation(t *testing.T) {
	result := buildDeletePollerMutation("99999")

	expects := []string{
		`mutation`,
		`deletePoller`,
		`id: "99999"`,
		`success`,
	}

	for _, exp := range expects {
		if !strings.Contains(result, exp) {
			t.Errorf("buildDeletePollerMutation() missing %q in:\n%s", exp, result)
		}
	}
}

func TestBuildPollerInputGQL_WithOptionalFields(t *testing.T) {
	retries := int64(3)
	cfg := &PollerConfig{
		Name:     "my-poller",
		Period:   300,
		Delay:    300,
		Interval: "1m",
		Retries:  &retries,
		Queries: []QueryConfig{
			{Namespace: "AWS/EC2"},
		},
	}

	result := buildPollerInputGQL(cfg, "us-west-2", "arn:aws:iam::123:role/test")

	expects := []string{
		`interval: "1m"`,
		`retries: "3"`,
		`cloudWatchMetricsConfig:`,
	}

	for _, exp := range expects {
		if !strings.Contains(result, exp) {
			t.Errorf("buildPollerInputGQL() missing %q in:\n%s", exp, result)
		}
	}
}

func TestBuildQueriesGQL_DimensionWithoutValue(t *testing.T) {
	queries := []QueryConfig{
		{
			Namespace: "AWS/EC2",
			Dimensions: []DimensionFilter{
				{Name: "InstanceId"},
			},
		},
	}

	result := buildQueriesGQL(queries)
	if !strings.Contains(result, `{name: "InstanceId"}`) {
		t.Errorf("expected dimension without value, got: %s", result)
	}
	if strings.Contains(result, `value:`) {
		t.Errorf("should not contain value field for empty value, got: %s", result)
	}
}

func TestBuildQueriesGQL_TagFilterWithoutValues(t *testing.T) {
	queries := []QueryConfig{
		{
			Namespace: "AWS/EC2",
			ResourceFilter: &ResourceFilter{
				TagFilters: []TagFilter{
					{Key: "Team"},
				},
			},
		},
	}

	result := buildQueriesGQL(queries)
	if !strings.Contains(result, `{key: "Team"}`) {
		t.Errorf("expected tag filter without values, got: %s", result)
	}
}
