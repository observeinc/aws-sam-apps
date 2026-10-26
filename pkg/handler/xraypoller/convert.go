package xraypoller

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

const (
	spanKindServer   = 2
	spanKindClient   = 3
	spanKindProducer = 4

	statusOK    = 1
	statusError = 2

	// DefaultAPIGatewayFilter limits collection to traces that touch API
	// Gateway. HTTP APIs (v2) do not emit X-Ray at all.
	DefaultAPIGatewayFilter = `service(id(type: "AWS::ApiGateway::Stage"))`
)

// Span is an OTLP span encoded the way the OTLP/JSON mapping expects int64
// fields: as decimal strings.
type Span struct {
	TraceID           string      `json:"traceId"`
	SpanID            string      `json:"spanId"`
	ParentSpanID      string      `json:"parentSpanId,omitempty"`
	Name              string      `json:"name"`
	Kind              int         `json:"kind"`
	StartTimeUnixNano string      `json:"startTimeUnixNano"`
	EndTimeUnixNano   string      `json:"endTimeUnixNano"`
	Attributes        []Attribute `json:"attributes,omitempty"`
	Status            Status      `json:"status"`
}

type Status struct {
	Code    int    `json:"code"`
	Message string `json:"message,omitempty"`
}

type Attribute struct {
	Key   string         `json:"key"`
	Value AttributeValue `json:"value"`
}

type AttributeValue struct {
	StringValue string `json:"stringValue,omitempty"`
	IntValue    string `json:"intValue,omitempty"`
}

// ResourceSpans groups spans that share a service name.
type ResourceSpans struct {
	Resource   Resource    `json:"resource"`
	ScopeSpans []ScopeSpan `json:"scopeSpans"`
}

type Resource struct {
	Attributes []Attribute `json:"attributes"`
}

type ScopeSpan struct {
	Scope Scope  `json:"scope"`
	Spans []Span `json:"spans"`
}

type Scope struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type serviceGroup struct {
	service string
	account string
	spans   []Span
}

// OTelTraceID converts an X-Ray trace id (1-{time}-{rand}) to the 32-hex
// OTLP trace id so backend spans that use the X-Ray propagator join it.
func OTelTraceID(xrayID string) (string, error) {
	parts := strings.Split(xrayID, "-")
	if len(parts) != 3 || parts[0] != "1" || len(parts[1]) != 8 || len(parts[2]) != 24 {
		return "", fmt.Errorf("invalid x-ray trace id %q", xrayID)
	}
	return parts[1] + parts[2], nil
}

// ResourceSpansFromDocuments converts X-Ray segment documents into OTLP
// resource spans. Inferred segments are dropped: they are X-Ray's synthetic
// view of an OTel-instrumented backend, and the real span arrives separately.
func ResourceSpansFromDocuments(xrayTraceID string, documents []string, region, scopeVersion string) ([]ResourceSpans, error) {
	traceID, err := OTelTraceID(xrayTraceID)
	if err != nil {
		return nil, err
	}

	groups := map[string]*serviceGroup{}
	var order []string
	for _, doc := range documents {
		var segment map[string]any
		if err := json.Unmarshal([]byte(doc), &segment); err != nil {
			return nil, fmt.Errorf("parse segment: %w", err)
		}
		if truthy(segment["inferred"]) {
			continue
		}
		addSegment(groups, &order, segment, traceID, 0, false)
	}

	out := make([]ResourceSpans, 0, len(order))
	for _, key := range order {
		group := groups[key]
		attrs := []Attribute{
			strAttr("service.name", group.service),
			strAttr("service.namespace", "aws-xray"),
			strAttr("cloud.provider", "aws"),
		}
		if region != "" {
			attrs = append(attrs, strAttr("cloud.region", region))
		}
		if group.account != "" {
			attrs = append(attrs, strAttr("cloud.account.id", group.account))
		}
		out = append(out, ResourceSpans{
			Resource: Resource{Attributes: attrs},
			ScopeSpans: []ScopeSpan{{
				Scope: Scope{Name: "xray-to-otlp-poller", Version: scopeVersion},
				Spans: group.spans,
			}},
		})
	}
	return out, nil
}

func addSegment(groups map[string]*serviceGroup, order *[]string, segment map[string]any, traceID string, rootEnd float64, isSub bool) {
	if truthy(segment["inferred"]) {
		return
	}
	spanID := str(segment["id"])
	if spanID == "" {
		return
	}
	if !isSub {
		if end, ok := num(segment["end_time"]); ok {
			rootEnd = end
		}
	}
	span := buildSpan(segment, traceID, isSub, rootEnd)
	service := serviceName(segment)
	key := service + "\x00" + accountID(segment)
	group, ok := groups[key]
	if !ok {
		group = &serviceGroup{service: service, account: accountID(segment)}
		groups[key] = group
		*order = append(*order, key)
	}
	group.spans = append(group.spans, span)

	for _, child := range slice(segment["subsegments"]) {
		childMap, ok := child.(map[string]any)
		if !ok {
			continue
		}
		addSegment(groups, order, childMap, traceID, rootEnd, true)
	}
}

func buildSpan(segment map[string]any, traceID string, isSub bool, rootEnd float64) Span {
	start, _ := num(segment["start_time"])
	end, hasEnd := num(segment["end_time"])
	if !hasEnd {
		end = start
	}
	startNS := epochNanos(start)
	endNS := epochNanos(end)
	if endNS == startNS {
		if rootEnd > end {
			endNS = epochNanos(rootEnd)
		} else {
			endNS = strconv.FormatInt(atoi(startNS)+1_000_000, 10)
		}
	}

	kind := spanKindServer
	if isSub {
		if str(segment["namespace"]) == "remote" {
			kind = spanKindClient
		} else {
			kind = spanKindProducer
		}
	}

	span := Span{
		TraceID:           traceID,
		SpanID:            str(segment["id"]),
		Name:              serviceName(segment),
		Kind:              kind,
		StartTimeUnixNano: startNS,
		EndTimeUnixNano:   endNS,
		Status:            statusOf(segment),
	}
	if parent := str(segment["parent_id"]); parent != "" {
		span.ParentSpanID = parent
	}

	span.Attributes = append(span.Attributes, httpAttributes(segment)...)
	if origin := str(segment["origin"]); origin != "" {
		span.Attributes = append(span.Attributes, strAttr("aws.xray.origin", origin))
	}
	span.Attributes = append(span.Attributes, strAttr("cloud.provider", "aws"))
	return span
}

func statusOf(segment map[string]any) Status {
	switch {
	case truthy(segment["fault"]):
		return Status{Code: statusError, Message: "fault"}
	case truthy(segment["error"]):
		return Status{Code: statusError, Message: "error"}
	default:
		return Status{Code: statusOK}
	}
}

func httpAttributes(segment map[string]any) []Attribute {
	httpMap := object(segment["http"])
	req := object(httpMap["request"])
	resp := object(httpMap["response"])
	var attrs []Attribute
	if url := str(req["url"]); url != "" {
		attrs = append(attrs, strAttr("http.url", url))
	}
	if method := str(req["method"]); method != "" {
		attrs = append(attrs, strAttr("http.method", method))
	}
	if status, ok := num(resp["status"]); ok {
		attrs = append(attrs, intAttr("http.status_code", int64(status)))
	}
	if ip := str(req["client_ip"]); ip != "" {
		attrs = append(attrs, strAttr("net.peer.ip", ip))
	}
	if ua := str(req["user_agent"]); ua != "" {
		attrs = append(attrs, strAttr("http.user_agent", ua))
	}
	awsMap := object(segment["aws"])
	if gw := object(awsMap["api_gateway"]); len(gw) > 0 {
		if id := str(gw["api_id"]); id != "" {
			attrs = append(attrs, strAttr("aws.api_gateway.api_id", id))
		}
		if stage := str(gw["stage"]); stage != "" {
			attrs = append(attrs, strAttr("aws.api_gateway.stage", stage))
		}
		if reqID := str(gw["request_id"]); reqID != "" {
			attrs = append(attrs, strAttr("aws.api_gateway.request_id", reqID))
		}
	}
	return attrs
}

func serviceName(segment map[string]any) string {
	if name := str(segment["name"]); name != "" {
		return name
	}
	if origin := str(segment["origin"]); origin != "" {
		return origin
	}
	return "aws-xray"
}

func accountID(segment map[string]any) string {
	return str(object(segment["aws"])["account_id"])
}

func strAttr(key, value string) Attribute {
	return Attribute{Key: key, Value: AttributeValue{StringValue: value}}
}

func intAttr(key string, value int64) Attribute {
	return Attribute{Key: key, Value: AttributeValue{IntValue: strconv.FormatInt(value, 10)}}
}

func epochNanos(ts float64) string {
	return strconv.FormatInt(int64(ts*1_000_000_000), 10)
}

func atoi(s string) int64 {
	n, _ := strconv.ParseInt(s, 10, 64)
	return n
}

func str(v any) string {
	s, _ := v.(string)
	return s
}

func num(v any) (float64, bool) {
	f, ok := v.(float64)
	return f, ok
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
