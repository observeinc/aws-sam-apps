package pollerconfigurator

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestBuildPollerInput_BasicFields(t *testing.T) {
	cfg := &PollerConfig{
		Name:     "test-poller",
		Period:   300,
		Delay:    60,
		Interval: "5m",
		Queries: []QueryConfig{
			{Namespace: "AWS/EC2", MetricNames: []string{"CPUUtilization"}},
		},
	}

	input := buildPollerInput(cfg, "ds-999", "us-east-1", "arn:aws:iam::999:role/observe")

	if input.Name != "test-poller" {
		t.Errorf("Name = %q, want %q", input.Name, "test-poller")
	}
	if input.DatastreamId != "ds-999" {
		t.Errorf("DatastreamId = %q, want %q", input.DatastreamId, "ds-999")
	}
	if input.Interval != "5m" {
		t.Errorf("Interval = %q, want %q", input.Interval, "5m")
	}
	if input.Retries != nil {
		t.Errorf("Retries = %v, want nil", input.Retries)
	}

	cw := input.CloudWatchMetricsConfig
	if cw == nil {
		t.Fatal("CloudWatchMetricsConfig is nil")
	}
	if cw.Period != "300" {
		t.Errorf("Period = %q, want %q", cw.Period, "300")
	}
	if cw.Delay != "60" {
		t.Errorf("Delay = %q, want %q", cw.Delay, "60")
	}
	if cw.Region != "us-east-1" {
		t.Errorf("Region = %q, want %q", cw.Region, "us-east-1")
	}
	if cw.AssumeRoleArn != "arn:aws:iam::999:role/observe" {
		t.Errorf("AssumeRoleArn = %q, want %q", cw.AssumeRoleArn, "arn:aws:iam::999:role/observe")
	}
	if len(cw.Queries) != 1 || cw.Queries[0].Namespace != "AWS/EC2" {
		t.Errorf("Queries = %+v, want 1 query with namespace AWS/EC2", cw.Queries)
	}
	if len(cw.Queries[0].MetricNames) != 1 || cw.Queries[0].MetricNames[0] != "CPUUtilization" {
		t.Errorf("MetricNames = %v, want [CPUUtilization]", cw.Queries[0].MetricNames)
	}
}

func TestBuildPollerInput_WithRetries(t *testing.T) {
	retries := int64(3)
	cfg := &PollerConfig{
		Name:     "my-poller",
		Period:   300,
		Delay:    300,
		Interval: "1m",
		Retries:  &retries,
		Queries:  []QueryConfig{{Namespace: "AWS/EC2"}},
	}

	input := buildPollerInput(cfg, "ds-999", "us-west-2", "arn:aws:iam::123:role/test")

	if input.Retries == nil || *input.Retries != "3" {
		t.Errorf("Retries = %v, want pointer to %q", input.Retries, "3")
	}
}

func TestBuildPollerInput_Dimensions(t *testing.T) {
	cfg := &PollerConfig{
		Period: 300, Delay: 300, Interval: "5m",
		Queries: []QueryConfig{
			{
				Namespace: "AWS/EC2",
				Dimensions: []DimensionFilter{
					{Name: "InstanceId", Value: "i-1234"},
					{Name: "AutoScalingGroupName"},
				},
			},
		},
	}

	input := buildPollerInput(cfg, "ds-1", "us-east-1", "arn:role")
	dims := input.CloudWatchMetricsConfig.Queries[0].Dimensions
	if len(dims) != 2 {
		t.Fatalf("got %d dimensions, want 2", len(dims))
	}
	if dims[0].Name != "InstanceId" || dims[0].Value != "i-1234" {
		t.Errorf("dims[0] = %+v, want {InstanceId, i-1234}", dims[0])
	}
	if dims[1].Name != "AutoScalingGroupName" || dims[1].Value != "" {
		t.Errorf("dims[1] = %+v, want {AutoScalingGroupName, (empty)}", dims[1])
	}
}

func TestBuildPollerInput_DimensionWithoutValue_OmittedInJSON(t *testing.T) {
	cfg := &PollerConfig{
		Period: 300, Delay: 300, Interval: "5m",
		Queries: []QueryConfig{
			{
				Namespace:  "AWS/EC2",
				Dimensions: []DimensionFilter{{Name: "InstanceId"}},
			},
		},
	}

	input := buildPollerInput(cfg, "ds-1", "us-east-1", "arn:role")
	data, err := json.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}

	var raw map[string]interface{}
	_ = json.Unmarshal(data, &raw)
	cw := raw["cloudWatchMetricsConfig"].(map[string]interface{})
	queries := cw["queries"].([]interface{})
	q := queries[0].(map[string]interface{})
	dims := q["dimensions"].([]interface{})
	d := dims[0].(map[string]interface{})

	if _, exists := d["value"]; exists {
		t.Error("expected empty value to be omitted from JSON")
	}
}

func TestBuildPollerInput_ResourceFilter(t *testing.T) {
	cfg := &PollerConfig{
		Period: 300, Delay: 300, Interval: "5m",
		Queries: []QueryConfig{
			{
				Namespace: "AWS/EC2",
				ResourceFilter: &ResourceFilter{
					ResourceType:  "AWS::EC2::Instance",
					Pattern:       "prod-*",
					DimensionName: "InstanceId",
					TagFilters: []TagFilter{
						{Key: "Environment", Values: []string{"prod", "staging"}},
						{Key: "Team"},
					},
				},
			},
		},
	}

	input := buildPollerInput(cfg, "ds-1", "us-east-1", "arn:role")
	rf := input.CloudWatchMetricsConfig.Queries[0].ResourceFilter
	if rf == nil {
		t.Fatal("ResourceFilter is nil")
	}
	if rf.ResourceType != "AWS::EC2::Instance" {
		t.Errorf("ResourceType = %q, want %q", rf.ResourceType, "AWS::EC2::Instance")
	}
	if rf.Pattern != "prod-*" {
		t.Errorf("Pattern = %q, want %q", rf.Pattern, "prod-*")
	}
	if rf.DimensionName != "InstanceId" {
		t.Errorf("DimensionName = %q, want %q", rf.DimensionName, "InstanceId")
	}
	if len(rf.TagFilters) != 2 {
		t.Fatalf("got %d tag filters, want 2", len(rf.TagFilters))
	}
	if rf.TagFilters[0].Key != "Environment" || len(rf.TagFilters[0].Values) != 2 {
		t.Errorf("TagFilters[0] = %+v, want {Environment, [prod staging]}", rf.TagFilters[0])
	}
	if rf.TagFilters[1].Key != "Team" || len(rf.TagFilters[1].Values) != 0 {
		t.Errorf("TagFilters[1] = %+v, want {Team, []}", rf.TagFilters[1])
	}
}

func TestBuildPollerInput_JSONRoundTrip(t *testing.T) {
	retries := int64(5)
	cfg := &PollerConfig{
		Name:     "roundtrip-test",
		Period:   600,
		Delay:    120,
		Interval: "10m",
		Retries:  &retries,
		Queries: []QueryConfig{
			{
				Namespace:   "AWS/Lambda",
				MetricNames: []string{"Invocations", "Errors"},
				Dimensions:  []DimensionFilter{{Name: "FunctionName", Value: "my-fn"}},
			},
		},
	}

	input := buildPollerInput(cfg, "ds-42", "eu-west-1", "arn:aws:iam::111:role/obs")
	data, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	var roundtripped pollerInput
	if err := json.Unmarshal(data, &roundtripped); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	if roundtripped.Name != "roundtrip-test" {
		t.Errorf("Name = %q after roundtrip", roundtripped.Name)
	}
	if roundtripped.CloudWatchMetricsConfig.Period != "600" {
		t.Errorf("Period = %q after roundtrip, want %q", roundtripped.CloudWatchMetricsConfig.Period, "600")
	}
	if roundtripped.Retries == nil || *roundtripped.Retries != "5" {
		t.Errorf("Retries = %v after roundtrip, want %q", roundtripped.Retries, "5")
	}
}

func TestBuildPollerInput_AttachResourceTags(t *testing.T) {
	base := func(attachResourceTags *bool) *PollerConfig {
		return &PollerConfig{
			Period: 300, Delay: 300, Interval: "5m",
			Queries:            []QueryConfig{{Namespace: "AWS/EC2"}},
			AttachResourceTags: attachResourceTags,
		}
	}

	t.Run("true", func(t *testing.T) {
		v := true
		input := buildPollerInput(base(&v), "ds-1", "us-east-1", "arn:role")
		cw := input.CloudWatchMetricsConfig
		if cw.AttachResourceTags == nil || !*cw.AttachResourceTags {
			t.Errorf("AttachResourceTags = %v, want true", cw.AttachResourceTags)
		}
		data, _ := json.Marshal(input)
		if !strings.Contains(string(data), `"attachResourceTags":true`) {
			t.Errorf("JSON missing attachResourceTags:true, got %s", data)
		}
	})

	t.Run("false", func(t *testing.T) {
		v := false
		input := buildPollerInput(base(&v), "ds-1", "us-east-1", "arn:role")
		cw := input.CloudWatchMetricsConfig
		if cw.AttachResourceTags == nil || *cw.AttachResourceTags {
			t.Errorf("AttachResourceTags = %v, want false", cw.AttachResourceTags)
		}
		data, _ := json.Marshal(input)
		if !strings.Contains(string(data), `"attachResourceTags":false`) {
			t.Errorf("JSON missing attachResourceTags:false, got %s", data)
		}
	})

	t.Run("nil omitted from JSON", func(t *testing.T) {
		input := buildPollerInput(base(nil), "ds-1", "us-east-1", "arn:role")
		if input.CloudWatchMetricsConfig.AttachResourceTags != nil {
			t.Errorf("AttachResourceTags = %v, want nil", *input.CloudWatchMetricsConfig.AttachResourceTags)
		}
		data, _ := json.Marshal(input)
		if strings.Contains(string(data), "attachResourceTags") {
			t.Errorf("JSON should not contain attachResourceTags, got %s", data)
		}
	})
}

func TestMutationConstants(t *testing.T) {
	tests := []struct {
		name     string
		mutation string
		want     []string
	}{
		{
			name:     "create",
			mutation: createPollerMutation,
			want:     []string{"mutation", "createPoller", "$workspaceId", "$poller", "id", "name"},
		},
		{
			name:     "update",
			mutation: updatePollerMutation,
			want:     []string{"mutation", "updatePoller", "$id", "$poller", "id", "name"},
		},
		{
			name:     "delete",
			mutation: deletePollerMutation,
			want:     []string{"mutation", "deletePoller", "$id", "success"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, w := range tt.want {
				if !contains(tt.mutation, w) {
					t.Errorf("%s mutation missing %q", tt.name, w)
				}
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
