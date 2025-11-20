package subscriber

import (
	"fmt"
	"sync/atomic"
)

// Response from our handler.
type Response struct {
	Discovery    *DiscoveryStats    `json:"discovery,omitempty"`
	Subscription *SubscriptionStats `json:"subscription,omitempty"`
	Cleanup      *CleanupStats      `json:"cleanup,omitempty"`
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
	// Cleanup stats are set if FullyPrune is enabled in the discovery request.
	Cleanup *CleanupStats `json:"cleanup,omitempty"`
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

// CleanupStats contains counters for cleanup operations.
type CleanupStats struct {
	// LogGroupsScanned tracks total number of log groups scanned.
	LogGroupsScanned Int64 `json:"logGroupsScanned,omitempty"`
	// SubscriptionsFound tracks number of our subscriptions found.
	SubscriptionsFound Int64 `json:"subscriptionsFound,omitempty"`
	// SubscriptionsKept tracks subscriptions that still match patterns.
	SubscriptionsKept Int64 `json:"subscriptionsKept,omitempty"`
	// SubscriptionsDeleted tracks subscriptions that were removed.
	SubscriptionsDeleted Int64 `json:"subscriptionsDeleted,omitempty"`
	// SubscriptionsWouldDelete tracks subscriptions that would be removed in dry-run mode.
	SubscriptionsWouldDelete Int64 `json:"subscriptionsWouldDelete,omitempty"`
}
