package request

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Handler processes batches of data towards same URI.
type Handler struct {
	URL       string
	Headers   map[string]string
	GzipLevel int
	Client    Doer
}

// Handle a batch of data.
func (h *Handler) Handle(ctx context.Context, body io.Reader) error {
	if h.GzipLevel != 0 {
		var buf bytes.Buffer
		gw := gzip.NewWriter(&buf)
		if _, err := io.Copy(gw, body); err != nil {
			return fmt.Errorf("failed to compress body: %w", err)
		}
		if err := gw.Close(); err != nil {
			return fmt.Errorf("failed to close compressed body: %w", err)
		}
		body = &buf
	}

	req, err := http.NewRequestWithContext(ctx, "POST", h.URL, body)
	if err != nil {
		return fmt.Errorf("failed to build request: %w", err)
	}

	if h.GzipLevel != 0 {
		req.Header.Set("Content-Encoding", "gzip")
	}
	for k, v := range h.Headers {
		req.Header.Set(k, v)
	}

	resp, err := h.Client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()
	if _, err := io.Copy(io.Discard, resp.Body); err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	switch resp.StatusCode {
	case http.StatusOK, http.StatusAccepted, http.StatusNoContent:
	default:
		return fmt.Errorf("%w: %s", ErrStatus, strings.ToLower(http.StatusText(resp.StatusCode)))
	}
	return nil
}
