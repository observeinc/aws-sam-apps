package xraypoller

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	targetPackage = "Tracing"
	maxPostTries  = 3
)

// Endpoint is the customer collect URL for OTLP/HTTP traces.
func Endpoint(customerID, domain string) string {
	domain = strings.TrimPrefix(domain, ".")
	return fmt.Sprintf("https://%s.collect.%s/v2/otel/v1/traces", customerID, domain)
}

type payload struct {
	ResourceSpans []ResourceSpans `json:"resourceSpans"`
}

func marshalPayload(resources []ResourceSpans) ([]byte, error) {
	body, err := json.Marshal(payload{ResourceSpans: resources})
	if err != nil {
		return nil, fmt.Errorf("encode otlp: %w", err)
	}
	return body, nil
}

// HTTPExporter posts OTLP/JSON to the Observe traces endpoint.
type HTTPExporter struct {
	URL    string
	Token  string
	Client *http.Client
	Sleep  func(context.Context, time.Duration) error
}

func (e *HTTPExporter) Export(ctx context.Context, body []byte) error {
	client := e.Client
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	var last error
	for attempt := 1; attempt <= maxPostTries; attempt++ {
		if attempt > 1 {
			if err := e.sleep(ctx, time.Duration(1<<(attempt-2))*time.Second); err != nil {
				return err
			}
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, e.URL, bytes.NewReader(body))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(e.Token))
		req.Header.Set("x-observe-target-package", targetPackage)

		resp, err := client.Do(req)
		if err != nil {
			last = err
			continue
		}
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		resp.Body.Close()
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return nil
		}
		last = fmt.Errorf("observe returned %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
		if resp.StatusCode != http.StatusTooManyRequests && resp.StatusCode < 500 {
			return last
		}
	}
	return last
}

func (e *HTTPExporter) sleep(ctx context.Context, d time.Duration) error {
	if e.Sleep != nil {
		return e.Sleep(ctx, d)
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
