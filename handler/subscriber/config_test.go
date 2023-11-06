package subscriber_test

import (
	"fmt"
	"testing"

	"github.com/observeinc/aws-sam-testing/handler/handlertest"
	"github.com/observeinc/aws-sam-testing/handler/subscriber"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestConfig(t *testing.T) {
	testcases := []struct {
		subscriber.Config
		ExpectError error
	}{
		{
			ExpectError: subscriber.ErrMissingCloudWatchLogsClient,
		},
		{
			Config: subscriber.Config{
				CloudWatchLogsClient: &handlertest.CloudWatchLogsClient{},
			},
			ExpectError: subscriber.ErrMissingQueue,
		},
		{
			Config: subscriber.Config{
				CloudWatchLogsClient: &handlertest.CloudWatchLogsClient{},
				Queue:                &MockQueue{},
			},
		},
	}

	for i, tc := range testcases {
		tc := tc
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			err := tc.Validate()
			if diff := cmp.Diff(err, tc.ExpectError, cmpopts.EquateErrors()); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}
