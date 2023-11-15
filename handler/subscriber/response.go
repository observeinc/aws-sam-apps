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

type DiscoveryStats struct {
	LogGroupCount Int64 `json:"logGroupCount,omitempty"`
	RequestCount  Int64 `json:"requestCount,omitempty"`
}

type SubscriptionStats struct {
	Deleted   Int64 `json:"deleted,omitempty"`
	Updated   Int64 `json:"updated,omitempty"`
	Skipped   Int64 `json:"skipped,omitempty"`
	Processed Int64 `json:"processed,omitempty"`
}
