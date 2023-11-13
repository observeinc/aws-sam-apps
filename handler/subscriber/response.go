package subscriber

import (
	"encoding/json"
	"expvar"
	"fmt"
)

// Response from our handler.
type Response struct {
	Discovery    *expvar.Map `json:"discovery,omitempty"`
	Subscription *expvar.Map `json:"subscription,omitempty"`
}

// MarshalJSON ensures we omit empty expvar Maps from result.
func (r *Response) MarshalJSON() ([]byte, error) {
	var response struct {
		Discovery    json.RawMessage `json:"discovery,omitempty"`
		Subscription json.RawMessage `json:"subscription,omitempty"`
	}

	if s := r.Discovery.String(); s != "{}" {
		response.Discovery = []byte(s)
	}

	if s := r.Subscription.String(); s != "{}" {
		response.Subscription = []byte(s)
	}

	data, err := json.Marshal(response)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response: %w", err)
	}
	return data, nil
}

// NewResponse returns an initialized response.
func NewResponse() *Response {
	return &Response{
		Discovery:    &expvar.Map{},
		Subscription: &expvar.Map{},
	}
}
