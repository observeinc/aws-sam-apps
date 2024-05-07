package s3http

import (
	"errors"
	"fmt"
	"net/url"
)

var (
	ErrInvalidDestination = errors.New("invalid destination URI")
	ErrMissingS3Client    = errors.New("missing S3 client")
)

type Config struct {
	DestinationURI string // HTTP URI to upload data to
	GetObjectAPIClient
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

	return errors.Join(errs...)
}
