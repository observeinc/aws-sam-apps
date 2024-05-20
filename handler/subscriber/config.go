package subscriber

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws/arn"
)

var (
	ErrMissingCloudWatchLogsClient = errors.New("missing CloudWatch Logs client")
	ErrMissingFilterName           = errors.New("filter name must be provided")
	ErrMissingDestinationARN       = errors.New("destination ARN must be provided if role ARN is set")
	ErrInvalidARN                  = errors.New("invalid ARN")
	ErrInvalidLogGroupName         = errors.New("invalid log group name substring")
	ErrInvalidRegexp               = errors.New("invalid regular expression")

	logGroupNameRe = regexp.MustCompile(`^[a-zA-Z0-9_\.\-\/]+$`)
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

	// RoleARN for subscription filter.
	// Only required if destination is a firehose delivery stream
	RoleARN *string

	// LogGroupNamePrefixes contains a list of prefixes which restricts the log
	// groups we operate on.
	LogGroupNamePrefixes []string
	// LogGroupNamePatterns contains a list of substrings which restricts the log
	// groups we operate on.
	LogGroupNamePatterns []string

	// ExcludeLogGroupNamePatterns allows filtering out log groups after they
	// have been retrieved from AWS
	ExcludeLogGroupNamePatterns []string

	// Number of concurrent workers. Defaults to number of CPUs.
	NumWorkers int
}

func (c *Config) Validate() error {
	var errs []error

	if c.CloudWatchLogsClient == nil {
		errs = append(errs, ErrMissingCloudWatchLogsClient)
	}

	if c.FilterName == "" {
		errs = append(errs, ErrMissingFilterName)
	}

	if c.DestinationARN != "" {
		if _, err := arn.Parse(c.DestinationARN); err != nil {
			errs = append(errs, fmt.Errorf("failed to parse destination: %w: %s", ErrInvalidARN, err))
		}
	}

	if c.RoleARN != nil {
		if c.DestinationARN == "" {
			errs = append(errs, ErrMissingDestinationARN)
		}

		roleARN, err := arn.Parse(*c.RoleARN)
		if err != nil || roleARN.Service != "iam" || !strings.HasPrefix(roleARN.Resource, "role/") {
			errs = append(errs, fmt.Errorf("failed to parse role: %w", ErrInvalidARN))
		}
	}

	for _, s := range append(c.LogGroupNamePatterns, c.LogGroupNamePrefixes...) {
		if !logGroupNameRe.MatchString(s) && s != "*" {
			errs = append(errs, fmt.Errorf("%w: %q", ErrInvalidLogGroupName, s))
		}
	}

	// Exclusion patterns may be regular expressions. This is because
	// `LogGroupNamePatterns` is a filter than can be pushed to AWS API,
	// whereas exclusion can only be performed on client side.
	for _, s := range c.ExcludeLogGroupNamePatterns {
		if _, err := regexp.Compile(s); err != nil {
			errs = append(errs, fmt.Errorf("%w: %q", ErrInvalidRegexp, s))
		}
	}

	return errors.Join(errs...)
}

func (c *Config) LogGroupFilter() FilterFunc {
	var includeRe, excludeRe *regexp.Regexp
	filterFunc := func(logGroupName string) bool {
		if excludeRe != nil && excludeRe.MatchString(logGroupName) {
			return false
		}
		if includeRe != nil && !includeRe.MatchString(logGroupName) {
			return false
		}
		return true
	}

	// build exclusion regex
	if len(c.ExcludeLogGroupNamePatterns) != 0 {
		excludeRe = regexp.MustCompile(strings.Join(c.ExcludeLogGroupNamePatterns, "|"))
	}

	// build inclusion regex
	var exprs []string

	for _, pattern := range c.LogGroupNamePatterns {
		if pattern == "*" {
			return filterFunc
		}
		exprs = append(exprs, pattern)
	}

	for _, prefix := range c.LogGroupNamePrefixes {
		if prefix == "*" {
			return filterFunc
		}
		exprs = append(exprs, fmt.Sprintf("^%s.*", prefix))
	}

	if len(exprs) != 0 {
		includeRe = regexp.MustCompile(strings.Join(exprs, "|"))
	}

	return filterFunc
}
