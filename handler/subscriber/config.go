package subscriber

import (
	"errors"

	"github.com/go-logr/logr"
)

var (
	ErrMissingCloudWatchLogsClient = errors.New("missing CloudWatch Logs client")
	ErrMissingQueue                = errors.New("missing queue")
)

type Config struct {
	CloudWatchLogsClient
	Queue

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

	return errors.Join(errs...)
}
