package tracing

import (
	"bytes"
	"context"
	"encoding/json"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/go-logr/logr"
)

type SQSWithContextHandler struct {
	handler lambda.Handler
}

// Compile time check our Handler implements lambda.Handler.
var _ lambda.Handler = SQSWithContextHandler{}

// Invoke checks if the incoming payload is from a SQS event and, if so,
// extracts the context from the SQS message and injects it into the context.
// This means that the spans created with this context will appear as children
// of a span in the discover request that created the SQS event
func (h SQSWithContextHandler) Invoke(ctx context.Context, payload []byte) ([]byte, error) {
	logger := logr.FromContextOrDiscard(ctx)
	logger.V(3).Info("Getting context from message attributes")
	var event events.SQSEvent
	dec := json.NewDecoder(bytes.NewReader(payload))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&event); err == nil {
		for _, record := range event.Records {
			ctx = NewSQSCarrier().Extract(ctx, record.MessageAttributes)
			break
		}
	}

	response, err := h.handler.Invoke(ctx, payload)
	if err != nil {
		return nil, err
	}

	return response, nil
}

// WrapHandlerSQSContext Provides a Handler which wraps an existing Handler while
// injecting the SQS context into the context if the payload is a SQS event.
func WrapHandlerSQSContext(handler lambda.Handler) lambda.Handler {
	return SQSWithContextHandler{handler: handler}
}
