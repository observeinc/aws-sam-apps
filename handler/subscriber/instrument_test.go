package subscriber_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/observeinc/aws-sam-apps/handler/handlertest"
	"github.com/observeinc/aws-sam-apps/handler/subscriber"
)

func TestInitTracing(t *testing.T) {
	testcases := []struct {
		ServiceName string
		ExpectError error
	}{
		{
			ServiceName: "test",
			ExpectError: nil,
		},
	}

	for i, tt := range testcases {
		tt := tt
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			t.Setenv("AWS_LAMBDA_FUNCTION_NAME", "test")
			ctx := context.Background()
			tracer, shutdownFn := subscriber.InitTracing(ctx, tt.ServiceName)
			tracer.Start(ctx, "test")
			err := shutdownFn(ctx)
			if diff := cmp.Diff(tt.ExpectError, err, cmpopts.EquateErrors()); diff != "" {
				t.Error(diff)
			}
		})
	}
}

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
