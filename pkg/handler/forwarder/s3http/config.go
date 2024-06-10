package s3http

import (
	"compress/gzip"
	"errors"
	"fmt"
	"net/http"
	"net/url"
)

var (
	ErrInvalidDestination   = errors.New("invalid destination URI")
	ErrMissingS3Client      = errors.New("missing S3 client")
	ErrUnsupportedGzipLevel = errors.New("unsupported compression level")
)

type Config struct {
	DestinationURI string // HTTP URI to upload data to
	GetObjectAPIClient
	HTTPClient *http.Client
	GzipLevel  *int
}

func (c *Config) Validate() error {
	var errs []error
	if c.DestinationURI == "" {
		errs = append(errs, fmt.Errorf("%w: %q", ErrInvalidDestination, c.DestinationURI))
	} else {
		u, err := url.ParseRequestURI(c.DestinationURI)
		switch {
		case err != nil:
			errs = append(errs, fmt.Errorf("%w: %w", ErrInvalidDestination, err))
		case u.Scheme == "https":
		default:
			errs = append(errs, fmt.Errorf("%w: scheme must be \"https\"", ErrInvalidDestination))
		}
	}

	if c.GetObjectAPIClient == nil {
		errs = append(errs, ErrMissingS3Client)
	}

	if c.GzipLevel != nil {
		if _, err := gzip.NewWriterLevel(nil, *c.GzipLevel); err != nil {
			errs = append(errs, fmt.Errorf("%w: %w", ErrUnsupportedGzipLevel, err))
		}
	}

	return errors.Join(errs...)
}
