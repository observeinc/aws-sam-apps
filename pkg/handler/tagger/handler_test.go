package tagger

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/go-logr/logr"
)

func TestHandleFirehose_JSON(t *testing.T) {
	t.Parallel()

	client := &mockTaggingClient{
		response: map[string]map[string]string{
			"i-123": {"Name": "web-1"},
		},
	}

	cache := NewTagCache(client, 5*time.Minute, t.TempDir(), logr.Discard())
	h := &Handler{
		Cache:        cache,
		OutputFormat: formatJSON,
		Logger:       logr.Discard(),
	}

	ctx := context.Background()
	event := events.KinesisFirehoseEvent{
		InvocationID: "test-invocation",
		Records: []events.KinesisFirehoseEventRecord{
			{
				RecordID: "rec-1",
				Data:     []byte(`{"namespace":"AWS/EC2","metric_name":"CPU","dimensions":{"InstanceId":"i-123"},"timestamp":1000,"value":{"count":1}}` + "\n"),
			},
		},
	}

	resp, err := h.HandleFirehose(ctx, event)
	if err != nil {
		t.Fatalf("HandleFirehose error: %v", err)
	}

	if len(resp.Records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(resp.Records))
	}

	rec := resp.Records[0]
	if rec.RecordID != "rec-1" {
		t.Errorf("RecordID=%q, want %q", rec.RecordID, "rec-1")
	}
	if rec.Result != events.KinesisFirehoseTransformedStateOk {
		t.Errorf("Result=%q, want Ok", rec.Result)
	}

	var metric cloudwatchMetricJSON
	lines := strings.TrimSpace(string(rec.Data))
	if err := json.Unmarshal([]byte(lines), &metric); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if metric.Tags["Name"] != "web-1" {
		t.Errorf("tag Name=%q, want %q", metric.Tags["Name"], "web-1")
	}
}

func TestHandleFirehose_MultipleRecords(t *testing.T) {
	t.Parallel()

	client := &mockTaggingClient{
		response: map[string]map[string]string{
			"i-111": {"Name": "s1"},
			"i-222": {"Name": "s2"},
		},
	}

	cache := NewTagCache(client, 5*time.Minute, t.TempDir(), logr.Discard())
	h := &Handler{
		Cache:        cache,
		OutputFormat: formatJSON,
		Logger:       logr.Discard(),
	}

	ctx := context.Background()
	event := events.KinesisFirehoseEvent{
		Records: []events.KinesisFirehoseEventRecord{
			{
				RecordID: "r1",
				Data:     []byte(`{"namespace":"AWS/EC2","dimensions":{"InstanceId":"i-111"},"timestamp":1000,"value":{"count":1}}` + "\n"),
			},
			{
				RecordID: "r2",
				Data:     []byte(`{"namespace":"AWS/EC2","dimensions":{"InstanceId":"i-222"},"timestamp":1000,"value":{"count":1}}` + "\n"),
			},
		},
	}

	resp, err := h.HandleFirehose(ctx, event)
	if err != nil {
		t.Fatalf("HandleFirehose error: %v", err)
	}

	if len(resp.Records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(resp.Records))
	}

	for _, rec := range resp.Records {
		if rec.Result != events.KinesisFirehoseTransformedStateOk {
			t.Errorf("record %s: Result=%q, want Ok", rec.RecordID, rec.Result)
		}
	}
}

func TestHandleFirehose_PreservesRecordID(t *testing.T) {
	t.Parallel()

	client := &mockTaggingClient{
		response: map[string]map[string]string{},
	}

	cache := NewTagCache(client, 5*time.Minute, t.TempDir(), logr.Discard())
	h := &Handler{
		Cache:        cache,
		OutputFormat: formatJSON,
		Logger:       logr.Discard(),
	}

	ctx := context.Background()
	event := events.KinesisFirehoseEvent{
		Records: []events.KinesisFirehoseEventRecord{
			{
				RecordID: "unique-id-123",
				Data:     []byte(`{"namespace":"Custom/X","dimensions":{},"timestamp":1000,"value":{"count":1}}` + "\n"),
			},
		},
	}

	resp, err := h.HandleFirehose(ctx, event)
	if err != nil {
		t.Fatalf("HandleFirehose error: %v", err)
	}

	if resp.Records[0].RecordID != "unique-id-123" {
		t.Errorf("RecordID=%q, want %q", resp.Records[0].RecordID, "unique-id-123")
	}
}
