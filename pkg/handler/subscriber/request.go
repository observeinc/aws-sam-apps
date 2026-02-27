package subscriber

import (
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
)

var ErrMalformedRequest = errors.New("malformed request")

// Request for our handler.
type Request struct {
	*SubscriptionRequest `json:"subscribe"`
	*DiscoveryRequest    `json:"discover"`
	*CleanupRequest      `json:"cleanup"`
}

// Validate verifies request is a union.
func (r *Request) Validate() error {
	if r == nil {
		return fmt.Errorf("%w: nil request", ErrMalformedRequest)
	}

	var count int
	if r.SubscriptionRequest != nil {
		count++
	}
	if r.DiscoveryRequest != nil {
		count++
	}
	if r.CleanupRequest != nil {
		count++
	}

	if count == 0 {
		return fmt.Errorf("%w: empty request", ErrMalformedRequest)
	}

	if count > 1 {
		return fmt.Errorf("%w: conflicting requests", ErrMalformedRequest)
	}

	return nil
}

// SubscriptionRequest contains a list of log groups to subscribe.
type SubscriptionRequest struct {
	// if provided, we can subscribe this set of log group names
	LogGroups []*LogGroup `json:"logGroups,omitempty"`
}

func NewSubscriptionRequestFromLogGroupsOutput(output *cloudwatchlogs.DescribeLogGroupsOutput) *SubscriptionRequest {
	var s SubscriptionRequest
	if output != nil {
		for _, logGroup := range output.LogGroups {
			if logGroup.LogGroupName != nil {
				s.LogGroups = append(s.LogGroups, &LogGroup{
					LogGroupName: *logGroup.LogGroupName,
				})
			}
		}
	}
	return &s
}

// DiscoveryRequest generates a list of log groups to subscribe.
type DiscoveryRequest struct {
	// optional filters
	LogGroupNamePatterns []*string `json:"logGroupNamePatterns,omitempty"`
	LogGroupNamePrefixes []*string `json:"logGroupNamePrefixes,omitempty"`

	// Limit when pagination list endpoint
	Limit *int32 `json:"limit,omitempty"`

	// Inline executes subscriptions inline with request
	// If not set, we default to however lambda is configured.
	Inline *bool `json:"inline,omitempty"`

	// FullyPrune if true, scans ALL log groups to find and remove subscriptions
	// that no longer match the current patterns. This is more expensive but ensures
	// stale subscriptions are cleaned up when patterns change (e.g., during stack updates).
	// If false (default), only log groups matching the current patterns are processed.
	FullyPrune bool `json:"fullyPrune,omitempty"`
}

// LogGroup represents the minimal viable info we need to be able to subscribe
// to a log group.
// Once we need to support linked accounts we'll likely need more than just a
// name.
type LogGroup struct {
	LogGroupName string `json:"logGroupName"`
}

// ToDescribeLogInputs computes the necessary describe-log-groups commands in order to unpack discovery request
// No attempt is made to dedupe log group names, since subscription is assumed to be idempotent.
func (d *DiscoveryRequest) ToDescribeLogInputs() (inputs []*cloudwatchlogs.DescribeLogGroupsInput) {
	if d == nil {
		return nil
	}

	for _, pattern := range d.LogGroupNamePatterns {
		if aws.ToString(pattern) == "*" {
			return []*cloudwatchlogs.DescribeLogGroupsInput{
				{
					Limit: d.Limit,
				},
			}
		}
		inputs = append(inputs, &cloudwatchlogs.DescribeLogGroupsInput{
			LogGroupNamePattern: pattern,
			Limit:               d.Limit,
		})
	}

	for _, prefix := range d.LogGroupNamePrefixes {
		if aws.ToString(prefix) == "*" {
			return []*cloudwatchlogs.DescribeLogGroupsInput{
				{
					Limit: d.Limit,
				},
			}
		}
		inputs = append(inputs, &cloudwatchlogs.DescribeLogGroupsInput{
			LogGroupNamePrefix: prefix,
			Limit:              d.Limit,
		})
	}

	return inputs
}

// CleanupRequest scans all log groups and removes subscriptions that no longer match the configured patterns.
type CleanupRequest struct {
	// DryRun if true, will only log what would be deleted without actually deleting
	DryRun bool `json:"dryRun,omitempty"`
	// DeleteAll if true, will delete all subscriptions regardless of whether they match patterns
	DeleteAll bool `json:"deleteAll,omitempty"`
}
