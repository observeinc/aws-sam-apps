package subscriber

import (
	"errors"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/go-logr/logr"
)

var (
	ErrMissingCloudWatchLogsClient = errors.New("missing CloudWatch Logs client")
	ErrMissingQueue                = errors.New("missing queue")
	ErrMissingFilterName           = errors.New("filter name must be provided if destination ARN is set")
	ErrMissingDestinationARN       = errors.New("destination ARN must be provided if role ARN is set")
	ErrInvalidARN                  = errors.New("invalid ARN")
)

type Config struct {
	CloudWatchLogsClient
	Queue

	// FilterName for subscription filters managed by this handler
	// Our handler will assume it manages all filters that have this name as a
	// prefix.
	FilterName string

	// FilterPattern for subscription filters
	FilterPattern string

	// DestinationARN to subscribe log groups to.
	// If empty, delete any subscription filters we manage.
	DestinationARN string
	// RoleARN for subscription filter
	RoleARN string

	Logger *logr.Logger
}

func (c *Config) Validate() error {
	var errs []error

	if c.CloudWatchLogsClient == nil {
		errs = append(errs, ErrMissingCloudWatchLogsClient)
	}

	if c.Queue == nil {
		errs = append(errs, ErrMissingQueue)
	}

	if c.FilterName == "" && c.DestinationARN != "" {
		errs = append(errs, ErrMissingFilterName)
	}

	if c.DestinationARN != "" {
		if _, err := arn.Parse(c.DestinationARN); err != nil {
			errs = append(errs, fmt.Errorf("failed to parse destination: %w: %s", ErrInvalidARN, err))
		}
	}

	if c.RoleARN != "" {
		if c.DestinationARN == "" {
			errs = append(errs, ErrMissingDestinationARN)
		}

		roleARN, err := arn.Parse(c.RoleARN)
		if err != nil || roleARN.Service != "iam" || strings.HasPrefix(roleARN.Resource, "role/") {
			errs = append(errs, fmt.Errorf("failed to parse role: %w", ErrInvalidARN))
		}
	}

	return errors.Join(errs...)
}
