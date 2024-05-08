package request

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/go-logr/logr"
	"github.com/hashicorp/go-retryablehttp"
)

var (
	ErrNoConfig   = errors.New("missing config")
	ErrMissingURL = errors.New("missing URL")
	ErrStatus     = errors.New("failed to upload: %w")
)

type Doer interface {
	Do(*http.Request) (*http.Response, error)
}

type BuilderConfig struct {
	URL string

	RetryWaitMin *time.Duration // Minimum time to wait on retry
	RetryWaitMax *time.Duration // Maximumum time to wait on retry
	RetryMax     *int           // Maximum number of retries

	HTTPClient *http.Client
	Logger     *logr.Logger
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

	return &Handler{
		URL:    u.String(),
		Client: b.Client,
	}
}

func NewBuilder(cfg *BuilderConfig) (*Builder, error) {
	if cfg == nil {
		return nil, ErrNoConfig
	}

	if _, err := url.Parse(cfg.URL); err != nil {
		return nil, fmt.Errorf("failed to parse base uri: %w", err)
	}

	client := retryablehttp.NewClient()

	if cfg.HTTPClient != nil {
		client.HTTPClient = cfg.HTTPClient
	}

	if cfg.RetryWaitMin != nil {
		client.RetryWaitMin = *cfg.RetryWaitMin
	}

	if cfg.RetryWaitMax != nil {
		client.RetryWaitMax = *cfg.RetryWaitMax
	}

	if cfg.RetryMax != nil {
		client.RetryMax = *cfg.RetryMax
	}

	logger := logr.Discard()
	if cfg.Logger != nil {
		logger = *cfg.Logger
	}

	client.Logger = &leveledLogger{logger}

	return &Builder{
		URL:    cfg.URL,
		Client: client.StandardClient(),
	}, nil
}
