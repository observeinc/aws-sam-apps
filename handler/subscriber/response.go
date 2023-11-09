package subscriber

// Response from our handler.
type Response struct {
	*SubscriptionResponse `json:"subscription"`
	*DiscoveryResponse    `json:"discovery"`
}

type DiscoveryResponse struct {
	// RequestCount tracks number of API requests
	RequestCount int
	// LogGroupCount tracks count of log groups retrieved from API
	LogGroupCount int
}

type SubscriptionResponse struct{}
