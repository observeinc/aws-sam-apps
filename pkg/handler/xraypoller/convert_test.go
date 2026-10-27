package xraypoller

import (
	"encoding/json"
	"testing"
)

func TestOTLPPayload(t *testing.T) {
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
				"end_time":   100.2,
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

	batch := NewTraceBatch()
	n, err := batch.Add("1-69fba214-225558552b59105d29f40f79", []string{string(doc)}, "us-west-2", "test")
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("spans: %d", n)
	}
	body, err := batch.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	payload := decodePayload(t, body)
	if len(payload.ResourceSpans) != 1 {
		t.Fatalf("resources: %d", len(payload.ResourceSpans))
	}
	rs := payload.ResourceSpans[0]
	if attr(rs.Resource, "service.name") != "my-api" {
		t.Fatalf("service: %s", attr(rs.Resource, "service.name"))
	}
	if attr(rs.Resource, "cloud.region") != "us-west-2" {
		t.Fatalf("region: %s", attr(rs.Resource, "cloud.region"))
	}
	if attr(rs.Resource, "service.namespace") != "aws-xray" {
		t.Fatalf("namespace: %s", attr(rs.Resource, "service.namespace"))
	}
	spans := rs.ScopeSpans[0].Spans
	if len(spans) != 2 {
		t.Fatalf("span count: %d", len(spans))
	}
	api := spans[0]
	if api.Name != "my-api" || api.Kind != 2 {
		t.Fatalf("root span: %+v", api)
	}
	if api.TraceID != "69fba214225558552b59105d29f40f79" {
		t.Fatalf("trace id: %s", api.TraceID)
	}
	if attr(api, "aws.api_gateway.stage") != "prod" {
		t.Fatalf("stage: %s", attr(api, "aws.api_gateway.stage"))
	}
	if attr(api, "aws.xray.origin") != "AWS::ApiGateway::Stage" {
		t.Fatalf("origin: %s", attr(api, "aws.xray.origin"))
	}
	if attr(api, "http.method") != "GET" {
		t.Fatalf("method: %s", attr(api, "http.method"))
	}

	ddb := spans[1]
	if ddb.Name != "DynamoDB" || ddb.Kind != 3 {
		t.Fatalf("subsegment: %+v", ddb)
	}
	if ddb.ParentSpanID != "abcdef1234567890" {
		t.Fatalf("parent: %s", ddb.ParentSpanID)
	}
}

func TestNestedSubsegmentInheritsParent(t *testing.T) {
	root := map[string]any{
		"id":         "aaaaaaaaaaaaaaaa",
		"name":       "my-api",
		"start_time": 100.0,
		"end_time":   100.5,
		"origin":     "AWS::ApiGateway::Stage",
		"subsegments": []any{
			map[string]any{
				"id":         "bbbbbbbbbbbbbbbb",
				"name":       "Lambda",
				"start_time": 100.1,
				"end_time":   100.4,
				"namespace":  "aws",
				"subsegments": []any{
					map[string]any{
						"id":         "cccccccccccccccc",
						"name":       "work",
						"start_time": 100.2,
						"end_time":   100.2,
					},
				},
			},
		},
	}
	doc, err := json.Marshal(root)
	if err != nil {
		t.Fatal(err)
	}
	batch := NewTraceBatch()
	n, err := batch.Add("1-69fba214-225558552b59105d29f40f79", []string{string(doc)}, "us-west-2", "test")
	if err != nil {
		t.Fatal(err)
	}
	if n != 3 {
		t.Fatalf("spans: %d", n)
	}
	spans := decodePayload(t, mustMarshal(t, batch)).ResourceSpans[0].ScopeSpans[0].Spans
	byName := map[string]otlpSpan{}
	for _, span := range spans {
		byName[span.Name] = span
	}
	if byName["my-api"].ParentSpanID != "" || byName["my-api"].Kind != 2 {
		t.Fatalf("root: %+v", byName["my-api"])
	}
	if byName["Lambda"].ParentSpanID != "aaaaaaaaaaaaaaaa" || byName["Lambda"].Kind != 3 {
		t.Fatalf("lambda: %+v", byName["Lambda"])
	}
	if byName["work"].ParentSpanID != "bbbbbbbbbbbbbbbb" || byName["work"].Kind != 1 {
		t.Fatalf("work: %+v", byName["work"])
	}
	payload := decodePayload(t, mustMarshal(t, batch))
	if attr(payload.ResourceSpans[0].Resource, "service.name") != "my-api" {
		t.Fatal("downstream subsegment took its own service name")
	}
	if len(payload.ResourceSpans) != 1 {
		t.Fatalf("resources: %d", len(payload.ResourceSpans))
	}
}

func TestUnknownNamespaceBecomesClient(t *testing.T) {
	doc := `{"id":"aaaaaaaaaaaaaaaa","name":"api/e2e","start_time":1,"end_time":2,"origin":"AWS::ApiGateway::Stage",` +
		`"subsegments":[{"id":"bbbbbbbbbbbbbbbb","name":"Mock","namespace":"mock","start_time":1,"end_time":1.5}]}`
	batch := NewTraceBatch()
	n, err := batch.Add("1-69fba214-225558552b59105d29f40f79", []string{doc}, "us-west-2", "test")
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("spans: %d", n)
	}
	var mock otlpSpan
	for _, span := range decodePayload(t, mustMarshal(t, batch)).ResourceSpans[0].ScopeSpans[0].Spans {
		if span.Name == "Mock" {
			mock = span
		}
	}
	if mock.Kind != 3 || mock.ParentSpanID != "aaaaaaaaaaaaaaaa" {
		t.Fatalf("mock: %+v", mock)
	}
	if attr(mock, "aws.xray.namespace") != "mock" {
		t.Fatalf("namespace attr: %q", attr(mock, "aws.xray.namespace"))
	}
}

func TestEmptyNamespaceIsLocal(t *testing.T) {
	doc := `{"id":"aaaaaaaaaaaaaaaa","name":"svc","start_time":1,"end_time":2,` +
		`"subsegments":[{"id":"bbbbbbbbbbbbbbbb","name":"work","namespace":"","start_time":1,"end_time":1.5}]}`
	batch := NewTraceBatch()
	n, err := batch.Add("1-69fba214-225558552b59105d29f40f79", []string{doc}, "us-west-2", "test")
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("spans: %d", n)
	}
	for _, span := range decodePayload(t, mustMarshal(t, batch)).ResourceSpans[0].ScopeSpans[0].Spans {
		if span.Name == "work" && (span.Kind != 1 || attr(span, "aws.xray.namespace") != "") {
			t.Fatalf("work: %+v", span)
		}
	}
}

func TestLargeIntegersSurviveNormalization(t *testing.T) {
	doc := `{"id":"aaaaaaaaaaaaaaaa","name":"svc","start_time":1,"end_time":2,` +
		`"aws":{"elastic_beanstalk":{"environment_name":"prod","version_label":"v1","deployment_id":9007199254740993}}}`
	batch := NewTraceBatch()
	if _, err := batch.Add("1-69fba214-225558552b59105d29f40f79", []string{doc}, "us-west-2", "test"); err != nil {
		t.Fatal(err)
	}
	resource := decodePayload(t, mustMarshal(t, batch)).ResourceSpans[0].Resource
	if got := attr(resource, "service.instance.id"); got != "9007199254740993" {
		t.Fatalf("service.instance.id: %q", got)
	}
}

func mustMarshal(t *testing.T, batch *TraceBatch) []byte {
	t.Helper()
	body, err := batch.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	return body
}

func TestInferredRootIsDropped(t *testing.T) {
	doc := `{"id":"abcdef1234567890","name":"backend","inferred":true,"start_time":1,"end_time":2}`
	batch := NewTraceBatch()
	n, err := batch.Add("1-69fba214-225558552b59105d29f40f79", []string{doc}, "us-east-1", "test")
	if err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("spans: %d", n)
	}
	body, err := batch.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	if body != nil {
		t.Fatalf("payload: %s", body)
	}
}

type otlpPayload struct {
	ResourceSpans []otlpResourceSpans `json:"resourceSpans"`
}

type otlpResourceSpans struct {
	Resource   otlpResource    `json:"resource"`
	ScopeSpans []otlpScopeSpan `json:"scopeSpans"`
}

type otlpResource struct {
	Attributes []otlpAttribute `json:"attributes"`
}

type otlpScopeSpan struct {
	Spans []otlpSpan `json:"spans"`
}

type otlpSpan struct {
	TraceID      string          `json:"traceId"`
	SpanID       string          `json:"spanId"`
	ParentSpanID string          `json:"parentSpanId"`
	Name         string          `json:"name"`
	Kind         int             `json:"kind"`
	Attributes   []otlpAttribute `json:"attributes"`
}

type otlpAttribute struct {
	Key   string `json:"key"`
	Value struct {
		StringValue string `json:"stringValue"`
	} `json:"value"`
}

func decodePayload(t *testing.T, body []byte) otlpPayload {
	t.Helper()
	var payload otlpPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatal(err)
	}
	return payload
}

func attr(holder interface{ attributes() []otlpAttribute }, key string) string {
	for _, attribute := range holder.attributes() {
		if attribute.Key == key {
			return attribute.Value.StringValue
		}
	}
	return ""
}

func (r otlpResource) attributes() []otlpAttribute { return r.Attributes }
func (s otlpSpan) attributes() []otlpAttribute     { return s.Attributes }
