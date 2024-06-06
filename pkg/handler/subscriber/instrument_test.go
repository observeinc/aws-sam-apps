package subscriber_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/observeinc/aws-sam-apps/pkg/handler/handlertest"
	"github.com/observeinc/aws-sam-apps/pkg/handler/subscriber"
)

func TestQueuePut(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		*subscriber.Request
		ExpectError error
	}{
		{
			Request:     &subscriber.Request{},
			ExpectError: nil,
		},
	}

	for i, tt := range testcases {
		tt := tt
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			t.Parallel()
			client := &handlertest.SQSClient{
				SendMessageFunc: func(_ context.Context, record *sqs.SendMessageInput, _ ...func(*sqs.Options)) (*sqs.SendMessageOutput, error) {
					_, found := record.MessageAttributes["b3"]
					if found != true {
						t.Error("b3 header not found")
					}
					return &sqs.SendMessageOutput{}, nil
				},
			}
			q, err := subscriber.NewQueue(client, "test")
			if err != nil {
				t.Fatal(err)
			}
			iq := subscriber.InstrumentQueue(*q)
			err = iq.Put(context.Background(), tt.Request)
			if diff := cmp.Diff(tt.ExpectError, err, cmpopts.EquateErrors()); diff != "" {
				t.Error(diff)
			}
		})
	}
}
