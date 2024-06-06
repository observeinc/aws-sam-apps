package subscriber_test

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/observeinc/aws-sam-apps/pkg/handler/subscriber"
)

func TestRequestMalformed(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		*subscriber.Request
		ExpectError error
	}{
		{
			Request:     &subscriber.Request{},
			ExpectError: subscriber.ErrMalformedRequest,
		},
		{
			Request: &subscriber.Request{
				SubscriptionRequest: &subscriber.SubscriptionRequest{},
				DiscoveryRequest:    &subscriber.DiscoveryRequest{},
			},
			ExpectError: subscriber.ErrMalformedRequest,
		},
		{
			Request: &subscriber.Request{
				SubscriptionRequest: &subscriber.SubscriptionRequest{},
			},
			ExpectError: nil,
		},
		{
			Request: &subscriber.Request{
				DiscoveryRequest: &subscriber.DiscoveryRequest{},
			},
			ExpectError: nil,
		},
	}

	for i, tt := range testcases {
		tt := tt
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			t.Parallel()

			err := tt.Request.Validate()
			if diff := cmp.Diff(tt.ExpectError, err, cmpopts.EquateErrors()); diff != "" {
				t.Error(diff)
			}
		})
	}
}
