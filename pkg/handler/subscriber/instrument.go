package subscriber

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/observeinc/aws-sam-apps/pkg/tracing"

	"github.com/go-logr/logr"
	"go.opentelemetry.io/contrib/propagators/b3"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

type InstrumentedHandler struct {
	*Handler
	trace.Tracer
}

func (h *InstrumentedHandler) HandleSQS(ctx context.Context, request events.SQSEvent) (response events.SQSEventResponse, err error) {
	logger := logr.FromContextOrDiscard(ctx)
	for _, record := range request.Records {
		var req Request
		var err error

		if err = json.Unmarshal([]byte(record.Body), &req); err == nil {
			_, err = h.HandleRequest(ctx, &req)
		}

		if err != nil {
			// SQS record will be under 256KB, should be ok to log
			logger.Error(err, "failed to process request", "body", record.Body)
			response.BatchItemFailures = append(response.BatchItemFailures, events.SQSBatchItemFailure{
				ItemIdentifier: record.MessageId,
			})
		}
	}
	return response, nil
}

func (h *InstrumentedHandler) HandleRequest(ctx context.Context, req *Request) (*Response, error) {
	ctx, span := h.Tracer.Start(ctx, "HandleRequest")
	var err error
	defer func() {
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()
	res, err := h.Handler.HandleRequest(ctx, req)
	return res, err
}

type InstrumentedSQSClient struct {
	Client     SQSClient
	Propagator propagation.TextMapPropagator
}

func (q *InstrumentedSQSClient) SendMessage(ctx context.Context, msg *sqs.SendMessageInput, _ ...func(*sqs.Options)) (*sqs.SendMessageOutput, error) {
	logger := logr.FromContextOrDiscard(ctx)
	logger.V(3).Info("Injecting context into message attributes")
	msg.MessageAttributes = make(map[string]types.MessageAttributeValue)
	if err := tracing.NewSQSCarrier().Inject(ctx, msg.MessageAttributes); err != nil {
		return nil, fmt.Errorf("failed to inject context into message attributes: %w", err)
	}
	logger.V(3).Info("sending message %s", msg.MessageBody)
	output, err := q.Client.SendMessage(ctx, msg)
	if err != nil {
		return nil, fmt.Errorf("failed to send message: %w", err)
	}
	return output, nil
}

func InstrumentQueue(q QueueWrapper) QueueWrapper {
	propagator := b3.New()
	instrumentedClient := &InstrumentedSQSClient{
		Client:     q.Client,
		Propagator: propagator,
	}
	q.Client = instrumentedClient
	return q
}
