package subscriber

import (
	"fmt"
	"sync/atomic"
)

// Response from our handler.
type Response struct {
	Discovery    *DiscoveryStats    `json:"discovery,omitempty"`
	Subscription *SubscriptionStats `json:"subscription,omitempty"`
}

// Int64 wraps around atomic.Int64 and provides marshalling method.
type Int64 struct {
	atomic.Int64
}

// MarshalJSON marshals as int.
func (i *Int64) MarshalJSON() ([]byte, error) {
	v := i.Load()
	return []byte(fmt.Sprintf("%d", v)), nil
}

// DiscoveryStats contains counters for discovering log groups.
type DiscoveryStats struct {
	// LogGroupCount tracks total number of log groups found.
	LogGroupCount Int64 `json:"logGroupCount,omitempty"`
	// RequestCount tracks total number of API calls to DescribeLogGroups.
	RequestCount Int64 `json:"requestCount,omitempty"`
	// Subscription stats are set if a request for inline subscription is set.
	Subscription *SubscriptionStats `json:"subscription,omitempty"`
}

// SubscriptionStats contains counters for subscription filter changes.
type SubscriptionStats struct {
	// Deleted subscription filters.
	Deleted Int64 `json:"deleted,omitempty"`
	// Updated subscription filters.
	Updated Int64 `json:"updated,omitempty"`
	// Skipped log groups.
	Skipped Int64 `json:"skipped,omitempty"`
	// Processed log groups.
	Processed Int64 `json:"processed,omitempty"`
}

// Add accumulates counters.
func (s *SubscriptionStats) Add(other *SubscriptionStats) {
	s.Deleted.Add(other.Deleted.Load())
	s.Updated.Add(other.Updated.Load())
	s.Skipped.Add(other.Skipped.Load())
	s.Processed.Add(other.Processed.Load())
}
