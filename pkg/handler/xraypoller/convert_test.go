package xraypoller

import (
	"encoding/json"
	"testing"
)

func TestOTelTraceID(t *testing.T) {
	got, err := OTelTraceID("1-69fba214-225558552b59105d29f40f79")
	if err != nil {
		t.Fatal(err)
	}
	if got != "69fba214225558552b59105d29f40f79" {
		t.Fatalf("trace id: %s", got)
	}
	if _, err := OTelTraceID("not-a-trace"); err == nil {
		t.Fatal("expected error")
	}
}

func TestResourceSpansFromDocuments(t *testing.T) {
	root := map[string]any{
		"id":         "abcdef1234567890",
		"name":       "my-api",
		"start_time": 100.0,
		"end_time":   100.5,
		"origin":     "AWS::ApiGateway::Stage",
		"http": map[string]any{
			"request":  map[string]any{"method": "GET", "url": "https://example.test/"},
			"response": map[string]any{"status": 200.0},
		},
		"aws": map[string]any{
			"account_id": "123456789012",
			"api_gateway": map[string]any{
				"api_id": "abc123",
				"stage":  "prod",
			},
		},
		"subsegments": []any{
			map[string]any{
				"id":         "1234abcd1234abcd",
				"name":       "DynamoDB",
				"start_time": 100.1,
				"end_time":   100.1,
				"namespace":  "remote",
				"parent_id":  "abcdef1234567890",
			},
			map[string]any{
				"id":        "ffffffffffffffff",
				"name":      "synthetic",
				"inferred":  true,
				"parent_id": "abcdef1234567890",
			},
		},
	}
	doc, err := json.Marshal(root)
	if err != nil {
		t.Fatal(err)
	}

	got, err := ResourceSpansFromDocuments("1-69fba214-225558552b59105d29f40f79", []string{string(doc)}, "us-west-2", "test")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("resource groups: %d", len(got))
	}

	api := findService(t, got, "my-api")
	apiSpan := api.ScopeSpans[0].Spans[0]
	if apiSpan.Kind != spanKindServer {
		t.Fatalf("root kind: %d", apiSpan.Kind)
	}
	if apiSpan.TraceID != "69fba214225558552b59105d29f40f79" {
		t.Fatalf("trace id: %s", apiSpan.TraceID)
	}
	if attr(apiSpan, "aws.api_gateway.stage") != "prod" {
		t.Fatalf("stage: %s", attr(apiSpan, "aws.api_gateway.stage"))
	}
	if attr(api.Resource, "cloud.region") != "us-west-2" {
		t.Fatalf("region: %s", attr(api.Resource, "cloud.region"))
	}

	ddb := findService(t, got, "DynamoDB")
	span := ddb.ScopeSpans[0].Spans[0]
	if span.Kind != spanKindClient {
		t.Fatalf("subsegment kind: %d", span.Kind)
	}
	if span.ParentSpanID != "abcdef1234567890" {
		t.Fatalf("parent: %s", span.ParentSpanID)
	}
	if span.EndTimeUnixNano == span.StartTimeUnixNano {
		t.Fatal("zero-duration subsegment was not extended to the root end")
	}
	if span.EndTimeUnixNano != epochNanos(100.5) {
		t.Fatalf("end: %s", span.EndTimeUnixNano)
	}
}

func TestInferredRootIsDropped(t *testing.T) {
	doc := `{"id":"abcdef1234567890","name":"backend","inferred":true,"start_time":1,"end_time":2}`
	got, err := ResourceSpansFromDocuments("1-69fba214-225558552b59105d29f40f79", []string{doc}, "us-east-1", "test")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("expected no spans, got %#v", got)
	}
}

func findService(t *testing.T, groups []ResourceSpans, name string) ResourceSpans {
	t.Helper()
	for _, group := range groups {
		if attr(group.Resource, "service.name") == name {
			return group
		}
	}
	t.Fatalf("missing service %s", name)
	return ResourceSpans{}
}

func attr(holder interface{ attributes() []Attribute }, key string) string {
	for _, attribute := range holder.attributes() {
		if attribute.Key == key {
			if attribute.Value.StringValue != "" {
				return attribute.Value.StringValue
			}
			return attribute.Value.IntValue
		}
	}
	return ""
}

func (r Resource) attributes() []Attribute { return r.Attributes }
func (s Span) attributes() []Attribute     { return s.Attributes }
