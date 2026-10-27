package xraypoller

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	coltracepb "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	tracepb "go.opentelemetry.io/proto/otlp/trace/v1"
	"google.golang.org/protobuf/proto"
)

const (
	defaultTargetPackage = "Tracing"
	// Stay under the exporter request cap and Observe's ingest limit so a
	// busy window is several posts instead of one that fails and pins the
	// cursor.
	defaultMaxExportBytes = 900_000
)

// Endpoint is the customer collect URL for OTLP/HTTP traces.
func Endpoint(accountID, domain string) string {
	domain = strings.TrimPrefix(domain, ".")
	return fmt.Sprintf("https://%s.collect.%s/v2/otel/v1/traces", accountID, domain)
}

// HTTPExporter posts OTLP/HTTP protobuf through the OpenTelemetry
// otlptracehttp client.
type HTTPExporter struct {
	URL           string
	Token         string
	TargetPackage string
	// MaxRequestBytes splits an export. Zero uses defaultMaxExportBytes.
	MaxRequestBytes int
	Retry           otlptracehttp.RetryConfig
	Client          *http.Client
}

func (e *HTTPExporter) Export(ctx context.Context, td ptrace.Traces) error {
	if td.ResourceSpans().Len() == 0 {
		return nil
	}
	parts, err := splitTraces(td, e.maxBytes())
	if err != nil {
		return err
	}
	client := otlptracehttp.NewClient(e.clientOptions()...)
	if err := client.Start(ctx); err != nil {
		return fmt.Errorf("start otlp exporter: %w", err)
	}
	defer func() {
		_ = client.Stop(context.WithoutCancel(ctx))
	}()
	for _, part := range parts {
		spans, err := resourceSpans(part)
		if err != nil {
			return err
		}
		if err := client.UploadTraces(ctx, spans); err != nil {
			return fmt.Errorf("export traces: %w", err)
		}
	}
	return nil
}

func (e *HTTPExporter) clientOptions() []otlptracehttp.Option {
	pkg := e.TargetPackage
	if pkg == "" {
		pkg = defaultTargetPackage
	}
	opts := []otlptracehttp.Option{
		otlptracehttp.WithEndpointURL(e.URL),
		otlptracehttp.WithHeaders(map[string]string{
			"Authorization":            "Bearer " + strings.TrimSpace(e.Token),
			"x-observe-target-package": pkg,
		}),
	}
	if e.Client != nil {
		opts = append(opts, otlptracehttp.WithHTTPClient(e.Client))
	}
	if e.Retry.Enabled {
		opts = append(opts, otlptracehttp.WithRetry(e.Retry))
	}
	return opts
}

func (e *HTTPExporter) maxBytes() int {
	if e.MaxRequestBytes > 0 {
		return e.MaxRequestBytes
	}
	return defaultMaxExportBytes
}

func resourceSpans(td ptrace.Traces) ([]*tracepb.ResourceSpans, error) {
	raw, err := (&ptrace.ProtoMarshaler{}).MarshalTraces(td)
	if err != nil {
		return nil, fmt.Errorf("encode otlp: %w", err)
	}
	var req coltracepb.ExportTraceServiceRequest
	if err := proto.Unmarshal(raw, &req); err != nil {
		return nil, fmt.Errorf("decode otlp: %w", err)
	}
	return req.ResourceSpans, nil
}

// splitTraces returns one batch when the payload fits, otherwise one batch
// per group of spans that fits in maxBytes. A single span larger than
// maxBytes is sent alone; it cannot be split further.
func splitTraces(td ptrace.Traces, maxBytes int) ([]ptrace.Traces, error) {
	if tracesSize(td) <= maxBytes {
		return []ptrace.Traces{td}, nil
	}
	var parts []ptrace.Traces
	cur := ptrace.NewTraces()
	flush := func() {
		if cur.ResourceSpans().Len() == 0 {
			return
		}
		parts = append(parts, cur)
		cur = ptrace.NewTraces()
	}
	resources := td.ResourceSpans()
	for i := 0; i < resources.Len(); i++ {
		rs := resources.At(i)
		scopes := rs.ScopeSpans()
		for j := 0; j < scopes.Len(); j++ {
			ss := scopes.At(j)
			spans := ss.Spans()
			for k := 0; k < spans.Len(); k++ {
				one := ptrace.NewTraces()
				addSpan(one, rs, ss, spans.At(k))
				if tracesSize(one) > maxBytes {
					flush()
					parts = append(parts, one)
					continue
				}
				if cur.ResourceSpans().Len() > 0 && tracesSize(cur)+tracesSize(one) > maxBytes {
					flush()
				}
				addSpan(cur, rs, ss, spans.At(k))
			}
		}
	}
	flush()
	if len(parts) == 0 {
		return []ptrace.Traces{td}, nil
	}
	return parts, nil
}

func addSpan(dst ptrace.Traces, srcRS ptrace.ResourceSpans, srcSS ptrace.ScopeSpans, span ptrace.Span) {
	rs := dst.ResourceSpans().AppendEmpty()
	srcRS.Resource().CopyTo(rs.Resource())
	rs.SetSchemaUrl(srcRS.SchemaUrl())
	ss := rs.ScopeSpans().AppendEmpty()
	srcSS.Scope().CopyTo(ss.Scope())
	ss.SetSchemaUrl(srcSS.SchemaUrl())
	span.CopyTo(ss.Spans().AppendEmpty())
}

func tracesSize(td ptrace.Traces) int {
	return (&ptrace.ProtoMarshaler{}).TracesSize(td)
}
