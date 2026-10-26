package xraypoller

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
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
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), "resourceSpans") {
			t.Errorf("body: %s", body)
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
		Sleep:  func(context.Context, time.Duration) error { return nil },
	}
	if err := exporter.Export(context.Background(), []byte(`{"resourceSpans":[]}`)); err != nil {
		t.Fatal(err)
	}
	if attempts != 2 {
		t.Fatalf("attempts: %d", attempts)
	}
}

func TestEndpoint(t *testing.T) {
	got := Endpoint("123", "observeinc.com")
	want := "https://123.collect.observeinc.com/v2/otel/v1/traces"
	if got != want {
		t.Fatalf("got %s", got)
	}
}
