package xraypoller

import (
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	coltracepb "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	"google.golang.org/protobuf/proto"
)

func TestHTTPExporterRetriesThenSucceeds(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if r.Header.Get("Authorization") != "Bearer secret-token" {
			t.Errorf("authorization: %s", r.Header.Get("Authorization"))
		}
		if r.Header.Get("x-observe-target-package") != "Tracing" {
			t.Errorf("package: %s", r.Header.Get("x-observe-target-package"))
		}
		if r.Header.Get("Content-Type") != "application/x-protobuf" {
			t.Errorf("content type: %s", r.Header.Get("Content-Type"))
		}
		if spanName(t, r) != "checkout" {
			t.Errorf("span: %s", spanName(t, r))
		}
		if attempts < 2 {
			http.Error(w, "busy", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	exporter := &HTTPExporter{
		URL:    server.URL,
		Token:  "secret-token",
		Client: server.Client(),
		Retry: otlptracehttp.RetryConfig{
			Enabled:         true,
			InitialInterval: time.Millisecond,
			MaxInterval:     time.Millisecond,
			MaxElapsedTime:  2 * time.Second,
		},
	}
	if err := exporter.Export(context.Background(), oneSpan("checkout")); err != nil {
		t.Fatal(err)
	}
	if attempts != 2 {
		t.Fatalf("attempts: %d", attempts)
	}
}

func TestHTTPExporterTargetPackageOverride(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-observe-target-package") != "Custom" {
			t.Errorf("package: %s", r.Header.Get("x-observe-target-package"))
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	exporter := &HTTPExporter{
		URL:           server.URL,
		Token:         "secret-token",
		TargetPackage: "Custom",
		Client:        server.Client(),
	}
	if err := exporter.Export(context.Background(), oneSpan("checkout")); err != nil {
		t.Fatal(err)
	}
}

func TestHTTPExporterBatchesBySize(t *testing.T) {
	posts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		posts++
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	td := ptrace.NewTraces()
	addNamedSpan(td, "one")
	addNamedSpan(td, "two")
	exporter := &HTTPExporter{
		URL:             server.URL,
		Token:           "secret-token",
		Client:          server.Client(),
		MaxRequestBytes: 1,
	}
	if err := exporter.Export(context.Background(), td); err != nil {
		t.Fatal(err)
	}
	if posts != 2 {
		t.Fatalf("posts: %d", posts)
	}
}

func TestEndpoint(t *testing.T) {
	got := Endpoint("123", "observeinc.com")
	want := "https://123.collect.observeinc.com/v2/otel/v1/traces"
	if got != want {
		t.Fatalf("got %s", got)
	}
}

func oneSpan(name string) ptrace.Traces {
	td := ptrace.NewTraces()
	addNamedSpan(td, name)
	return td
}

func addNamedSpan(td ptrace.Traces, name string) {
	rs := td.ResourceSpans().AppendEmpty()
	rs.Resource().Attributes().PutStr("service.name", name)
	span := rs.ScopeSpans().AppendEmpty().Spans().AppendEmpty()
	span.SetName(name)
}

func spanName(t *testing.T, r *http.Request) string {
	t.Helper()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		t.Fatal(err)
	}
	if r.Header.Get("Content-Encoding") == "gzip" {
		zr, err := gzip.NewReader(bytes.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		body, err = io.ReadAll(zr)
		if err != nil {
			t.Fatal(err)
		}
	}
	var req coltracepb.ExportTraceServiceRequest
	if err := proto.Unmarshal(body, &req); err != nil {
		t.Fatal(err)
	}
	if len(req.ResourceSpans) == 0 || len(req.ResourceSpans[0].ScopeSpans) == 0 || len(req.ResourceSpans[0].ScopeSpans[0].Spans) == 0 {
		t.Fatal("empty export")
	}
	return req.ResourceSpans[0].ScopeSpans[0].Spans[0].GetName()
}
