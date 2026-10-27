package xraypoller

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"

	"github.com/observeinc/aws-sam-apps/pkg/xrayotel/telemetry"
	"github.com/observeinc/aws-sam-apps/pkg/xrayotel/translator"
)

const (
	// DefaultAPIGatewayFilter limits collection to traces that touch API
	// Gateway. HTTP APIs (v2) do not emit X-Ray at all.
	DefaultAPIGatewayFilter = `service(id(type: "AWS::ApiGateway::Stage"))`

	scopeName = "xray-to-otlp-poller"
)

// TraceBatch collects converted traces for the OTLP exporter.
type TraceBatch struct {
	traces ptrace.Traces
}

// NewTraceBatch returns an empty batch.
func NewTraceBatch() *TraceBatch {
	return &TraceBatch{traces: ptrace.NewTraces()}
}

// Add converts one X-Ray trace's segment documents and appends them.
// Inferred segments are dropped before translation. The returned count is
// the number of spans added.
func (b *TraceBatch) Add(xrayTraceID string, documents []string, region, scopeVersion string) (int, error) {
	pending := ptrace.NewTraces()
	added := 0
	for _, doc := range documents {
		prepared, skip, err := prepareDocument(doc, xrayTraceID)
		if err != nil {
			return 0, err
		}
		if skip {
			continue
		}
		td, _, err := translator.ToTraces(prepared.body, telemetry.Nop{})
		if err != nil {
			return 0, err
		}
		enrich(td, prepared, region, scopeVersion)
		added += countSpans(td)
		td.ResourceSpans().MoveAndAppendTo(pending.ResourceSpans())
	}
	pending.ResourceSpans().MoveAndAppendTo(b.traces.ResourceSpans())
	return added, nil
}

// Traces returns the converted spans. The caller owns the value.
func (b *TraceBatch) Traces() ptrace.Traces {
	return b.traces
}

// Marshal encodes the batch as OTLP/JSON. An empty batch returns nil.
func (b *TraceBatch) Marshal() ([]byte, error) {
	if b.traces.ResourceSpans().Len() == 0 {
		return nil, nil
	}
	body, err := (&ptrace.JSONMarshaler{}).MarshalTraces(b.traces)
	if err != nil {
		return nil, fmt.Errorf("encode otlp: %w", err)
	}
	return body, nil
}

// preparedDocument is a segment document ready for the translator. root is
// the decoded form of body, and namespaces holds the original namespace of
// each subsegment id whose namespace was rewritten.
type preparedDocument struct {
	body       []byte
	root       map[string]any
	namespaces map[string]string
}

// prepareDocument drops an inferred document, removes inferred subsegments,
// maps namespaces the translator rejects, and fills trace_id from the
// BatchGetTraces id when the document omits it.
func prepareDocument(doc, xrayTraceID string) (preparedDocument, bool, error) {
	var root map[string]any
	dec := json.NewDecoder(strings.NewReader(doc))
	// Annotations and metadata can hold integers beyond float64 precision.
	dec.UseNumber()
	if err := dec.Decode(&root); err != nil {
		return preparedDocument{}, false, fmt.Errorf("parse segment: %w", err)
	}
	if truthy(root["inferred"]) {
		return preparedDocument{}, true, nil
	}
	stripInferred(root)
	namespaces := map[string]string{}
	normalizeNamespaces(root, namespaces)
	if str(root["trace_id"]) == "" {
		if xrayTraceID == "" {
			return preparedDocument{}, false, fmt.Errorf("missing trace id")
		}
		root["trace_id"] = xrayTraceID
	}
	body, err := json.Marshal(root)
	if err != nil {
		return preparedDocument{}, false, err
	}
	return preparedDocument{body: body, root: root, namespaces: namespaces}, false, nil
}

func stripInferred(seg map[string]any) {
	children := slice(seg["subsegments"])
	if len(children) == 0 {
		return
	}
	kept := make([]any, 0, len(children))
	for _, child := range children {
		m, ok := child.(map[string]any)
		if !ok || truthy(m["inferred"]) {
			continue
		}
		stripInferred(m)
		kept = append(kept, m)
	}
	if len(kept) == 0 {
		delete(seg, "subsegments")
		return
	}
	seg["subsegments"] = kept
}

// normalizeNamespaces makes every subsegment namespace one the translator
// accepts: absent, aws, remote, or local. An empty namespace is removed, and
// any other value, such as API Gateway's "mock" integration, becomes
// "remote" with the original recorded in replaced by subsegment id.
func normalizeNamespaces(seg map[string]any, replaced map[string]string) {
	for _, child := range slice(seg["subsegments"]) {
		m, ok := child.(map[string]any)
		if !ok {
			continue
		}
		switch ns := m["namespace"]; ns {
		case "aws", "remote", "local":
		case nil, "":
			delete(m, "namespace")
		default:
			replaced[str(m["id"])] = fmt.Sprint(ns)
			m["namespace"] = "remote"
		}
		normalizeNamespaces(m, replaced)
	}
}

// enrich stamps resource and span attributes the translator does not set.
func enrich(td ptrace.Traces, doc preparedDocument, region, scopeVersion string) {
	resources := td.ResourceSpans()
	for i := 0; i < resources.Len(); i++ {
		rs := resources.At(i)
		attrs := rs.Resource().Attributes()
		if region != "" {
			attrs.PutStr("cloud.region", region)
		}
		attrs.PutStr("service.namespace", "aws-xray")
		scopes := rs.ScopeSpans()
		for j := 0; j < scopes.Len(); j++ {
			scope := scopes.At(j)
			scope.Scope().SetName(scopeName)
			if scopeVersion != "" {
				scope.Scope().SetVersion(scopeVersion)
			}
			stampSegment(scope.Spans(), doc.root, doc.namespaces)
		}
	}
}

func stampSegment(spans ptrace.SpanSlice, seg map[string]any, namespaces map[string]string) {
	if span, ok := findSpan(spans, str(seg["id"])); ok {
		if origin := str(seg["origin"]); origin != "" {
			span.Attributes().PutStr("aws.xray.origin", origin)
		}
		if ns, ok := namespaces[str(seg["id"])]; ok {
			span.Attributes().PutStr("aws.xray.namespace", ns)
		}
		gw := object(object(seg["aws"])["api_gateway"])
		if id := str(gw["api_id"]); id != "" {
			span.Attributes().PutStr("aws.api_gateway.api_id", id)
		}
		if stage := str(gw["stage"]); stage != "" {
			span.Attributes().PutStr("aws.api_gateway.stage", stage)
		}
		if reqID := str(gw["request_id"]); reqID != "" {
			span.Attributes().PutStr("aws.api_gateway.request_id", reqID)
		}
	}
	for _, child := range slice(seg["subsegments"]) {
		childMap, ok := child.(map[string]any)
		if !ok {
			continue
		}
		stampSegment(spans, childMap, namespaces)
	}
}

func findSpan(spans ptrace.SpanSlice, hexID string) (ptrace.Span, bool) {
	raw, err := hex.DecodeString(hexID)
	if err != nil || len(raw) != 8 {
		return ptrace.Span{}, false
	}
	var id pcommon.SpanID
	copy(id[:], raw)
	for i := 0; i < spans.Len(); i++ {
		if spans.At(i).SpanID() == id {
			return spans.At(i), true
		}
	}
	return ptrace.Span{}, false
}

func countSpans(td ptrace.Traces) int {
	n := 0
	resources := td.ResourceSpans()
	for i := 0; i < resources.Len(); i++ {
		scopes := resources.At(i).ScopeSpans()
		for j := 0; j < scopes.Len(); j++ {
			n += scopes.At(j).Spans().Len()
		}
	}
	return n
}

func str(v any) string {
	s, _ := v.(string)
	return s
}

func truthy(v any) bool {
	b, _ := v.(bool)
	return b
}

func object(v any) map[string]any {
	m, _ := v.(map[string]any)
	if m == nil {
		return map[string]any{}
	}
	return m
}

func slice(v any) []any {
	s, _ := v.([]any)
	return s
}
