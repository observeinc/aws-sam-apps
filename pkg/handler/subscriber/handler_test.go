package subscriber_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/google/go-cmp/cmp"

	"github.com/observeinc/aws-sam-apps/pkg/handler/subscriber"
	"github.com/observeinc/aws-sam-apps/pkg/testing/awstest"
)

func TestHandleSQS(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		Event  events.SQSEvent
		Expect events.SQSEventResponse
	}{
		{
			Event: events.SQSEvent{
				Records: []events.SQSMessage{
					{
						MessageId: "invalid request",
						Body:      "",
					},
				},
			},
			Expect: events.SQSEventResponse{
				BatchItemFailures: []events.SQSBatchItemFailure{
					{ItemIdentifier: "invalid request"},
				},
			},
		},
		{
			Event: events.SQSEvent{
				Records: []events.SQSMessage{
					{
						MessageId: "ok",
						Body:      "{\"subscribe\": {\"logGroups\":[{\"logGroupName\":\"/aws/hello\"}]}}",
					},
				},
			},
			Expect: events.SQSEventResponse{},
		},
	}

	for i, tt := range testcases {
		tt := tt
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			t.Parallel()

			s, err := subscriber.New(&subscriber.Config{
				CloudWatchLogsClient: &awstest.CloudWatchLogsClient{},
				FilterName:           "test",
				LogGroupNamePrefixes: []string{"*"},
			})
			if err != nil {
				t.Fatal(err)
			}

			resp, err := s.HandleSQS(context.Background(), tt.Event)
			if err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(resp, tt.Expect); diff != "" {
				t.Error(diff)
			}
		})
	}
}
