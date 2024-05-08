package request

import (
	"errors"
	"net/http"
	"net/url"
)

var (
	ErrNoConfig   = errors.New("missing config")
	ErrMissingURL = errors.New("missing URL")
	ErrStatus     = errors.New("failed to upload: %w")
)

type Doer interface {
	Do(*http.Request) (*http.Response, error)
}

// Builder shares an HTTP client across request handlers.
type Builder struct {
	Client Doer
	URL    string
}

func (b *Builder) With(tags map[string]string) *Handler {
	// convert tags into query parameters
	values := make(url.Values, len(tags))
	for k, v := range tags {
		values.Add(k, v)
	}

	u, _ := url.Parse(b.URL)
	u.RawQuery = values.Encode()

	client := b.Client
	if client == nil {
		client = http.DefaultClient
	}

	return &Handler{
		URL:    u.String(),
		Client: client,
	}
}
