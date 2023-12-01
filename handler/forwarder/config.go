package forwarder

import (
	"errors"
	"fmt"
	"net/url"

	"github.com/go-logr/logr"
)

var (
	ErrInvalidDestination          = errors.New("invalid destination URI")
	ErrMissingS3Client             = errors.New("missing S3 client")
	ErrInvalidContentTypeOverrides = errors.New("invalid content type overrides")
)

type Config struct {
	DestinationURI       string   // S3 URI to write messages and copy files to
	LogPrefix            string   // prefix used when writing SQS messages to S3
	MaxFileSize          int64    // maximum file size in bytes for the files to be processed
	ContentTypeOverrides []string // list of key pair values containing regular expressions to content type values
	SourceBucketNames    []string
	S3Client             S3Client
	Logger               *logr.Logger
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

	if _, err := NewContentTypeOverrides(c.ContentTypeOverrides, defaultDelimiter); err != nil {
		errs = append(errs, fmt.Errorf("%w: %w", ErrInvalidContentTypeOverrides, err))
	}

	if c.S3Client == nil {
		errs = append(errs, ErrMissingS3Client)
	}

	return errors.Join(errs...)
}
