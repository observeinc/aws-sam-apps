package filedropper

import (
	"errors"
	"fmt"
	"net/url"

	"github.com/go-logr/logr"
)

var (
	ErrInvalidDestination = errors.New("invalid destination URI")
	ErrMissingS3Client    = errors.New("missing S3 client")
)

type Config struct {
	DestinationURI string
	S3Client       S3Client
	Logger         *logr.Logger
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
		case u.Scheme != "s3":
			errs = append(errs, fmt.Errorf("%w: scheme must be \"s3\"", ErrInvalidDestination))
		}
	}

	if c.S3Client == nil {
		errs = append(errs, ErrMissingS3Client)
	}

	return errors.Join(errs...)
}
