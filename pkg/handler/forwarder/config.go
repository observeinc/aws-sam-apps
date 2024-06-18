package forwarder

import (
	"errors"
	"fmt"
	"net/url"
	"time"
)

var (
	ErrInvalidDestination = errors.New("invalid destination URI")
	ErrInvalidFilter      = errors.New("invalid source filter")
	ErrMissingS3Client    = errors.New("missing S3 client")
	ErrPresetNotFound     = errors.New("not found")
)

type Config struct {
	DestinationURI     string // S3 URI to write messages and copy files to
	MaxFileSize        int64  // maximum file size in bytes for the files to be processed
	SourceBucketNames  []string
	SourceObjectKeys   []string
	Override           Override
	S3Client           S3Client
	GetTime            func() *time.Time
	MaxConcurrentTasks int // fan out limit
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
		case u.Scheme == "s3":
		case u.Scheme == "https":
		default:
			errs = append(errs, fmt.Errorf("%w: scheme must be \"s3\" or \"https\"", ErrInvalidDestination))
		}
	}

	if _, err := NewObjectFilter(c.SourceBucketNames, c.SourceObjectKeys); err != nil {
		errs = append(errs, fmt.Errorf("%w: %w", ErrInvalidFilter, err))
	}

	if c.S3Client == nil {
		errs = append(errs, ErrMissingS3Client)
	}

	return errors.Join(errs...)
}
