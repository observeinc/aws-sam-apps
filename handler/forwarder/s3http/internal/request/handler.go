package request

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Handler processes batches of data towards same URI.
type Handler struct {
	URL    string
	Client Doer
}

// Handle a batch of data.
func (r *Handler) Handle(ctx context.Context, body io.Reader) error {
	req, err := http.NewRequestWithContext(ctx, "POST", r.URL, body)
	if err != nil {
		return fmt.Errorf("failed to build request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-ndjson")

	resp, err := r.Client.Do(req)
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
