package tagger

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/go-logr/logr"
)

func TestEnrichJSON(t *testing.T) {
	t.Parallel()

	client := &mockTaggingClient{
		response: map[string]map[string]string{
			"i-0abcd1234": {"Name": "web-server-1", "Environment": "prod"},
		},
	}

	cache := NewTagCache(client, 5*time.Minute, t.TempDir(), logr.Discard())
	ctx := context.Background()

	input := `{"metric_stream_name":"test","account_id":"123456789012","region":"us-east-1","namespace":"AWS/EC2","metric_name":"CPUUtilization","dimensions":{"InstanceId":"i-0abcd1234"},"timestamp":1611929698000,"value":{"min":0.0,"max":1.0,"count":1,"sum":1.0},"unit":"Percent"}
`

	result, err := enrichJSON(ctx, []byte(input), cache)
	if err != nil {
		t.Fatalf("enrichJSON error: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(result)), "\n")
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}

	var metric cloudwatchMetricJSON
	if err := json.Unmarshal([]byte(lines[0]), &metric); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}

	if metric.Tags == nil {
		t.Fatal("expected tags to be set")
	}
	if metric.Tags["Name"] != "web-server-1" {
		t.Errorf("Name=%q, want %q", metric.Tags["Name"], "web-server-1")
	}
	if metric.Tags["Environment"] != "prod" {
		t.Errorf("Environment=%q, want %q", metric.Tags["Environment"], "prod")
	}
	if metric.Namespace != "AWS/EC2" {
		t.Errorf("Namespace=%q, want %q", metric.Namespace, "AWS/EC2")
	}
}

func TestEnrichJSON_UnknownNamespace(t *testing.T) {
	t.Parallel()

	client := &mockTaggingClient{
		response: map[string]map[string]string{},
	}

	cache := NewTagCache(client, 5*time.Minute, t.TempDir(), logr.Discard())
	ctx := context.Background()

	input := `{"namespace":"Custom/MyApp","metric_name":"Requests","dimensions":{"AppId":"app-1"},"timestamp":1611929698000,"value":{"count":1}}
`

	result, err := enrichJSON(ctx, []byte(input), cache)
	if err != nil {
		t.Fatalf("enrichJSON error: %v", err)
	}

	var metric cloudwatchMetricJSON
	if err := json.Unmarshal([]byte(strings.TrimSpace(string(result))), &metric); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}

	if metric.Tags != nil {
		t.Errorf("expected nil tags for unknown namespace, got %v", metric.Tags)
	}
}

func TestEnrichJSON_MultipleRecords(t *testing.T) {
	t.Parallel()

	client := &mockTaggingClient{
		response: map[string]map[string]string{
			"i-111": {"Name": "server-1"},
			"i-222": {"Name": "server-2"},
		},
	}

	cache := NewTagCache(client, 5*time.Minute, t.TempDir(), logr.Discard())
	ctx := context.Background()

	input := `{"namespace":"AWS/EC2","metric_name":"CPUUtilization","dimensions":{"InstanceId":"i-111"},"timestamp":1000,"value":{"count":1}}
{"namespace":"AWS/EC2","metric_name":"NetworkIn","dimensions":{"InstanceId":"i-222"},"timestamp":1000,"value":{"count":1}}
`

	result, err := enrichJSON(ctx, []byte(input), cache)
	if err != nil {
		t.Fatalf("enrichJSON error: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(result)), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}

	var m1, m2 cloudwatchMetricJSON
	json.Unmarshal([]byte(lines[0]), &m1)
	json.Unmarshal([]byte(lines[1]), &m2)

	if m1.Tags["Name"] != "server-1" {
		t.Errorf("record 1: Name=%q, want %q", m1.Tags["Name"], "server-1")
	}
	if m2.Tags["Name"] != "server-2" {
		t.Errorf("record 2: Name=%q, want %q", m2.Tags["Name"], "server-2")
	}
}

func TestEnrichJSON_InvalidJSON(t *testing.T) {
	t.Parallel()

	client := &mockTaggingClient{
		response: map[string]map[string]string{},
	}

	cache := NewTagCache(client, 5*time.Minute, t.TempDir(), logr.Discard())
	ctx := context.Background()

	input := `not valid json
`

	result, err := enrichJSON(ctx, []byte(input), cache)
	if err != nil {
		t.Fatalf("enrichJSON error: %v", err)
	}

	if !strings.Contains(string(result), "not valid json") {
		t.Error("invalid JSON should be passed through unchanged")
	}
}
