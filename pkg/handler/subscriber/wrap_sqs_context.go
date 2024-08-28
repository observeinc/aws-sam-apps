package subscriber

import (
	"bytes"
	"context"
	"encoding/json"
	"reflect"

	"github.com/aws/aws-lambda-go/events"

	"github.com/aws/aws-lambda-go/lambda"
)

type SQSCheckerWrappedHandler struct {
	handler lambda.Handler
}

// Compile time check our Handler implements lambda.Handler.
var _ lambda.Handler = SQSCheckerWrappedHandler{}

// Invoke adds OTel span surrounding customer Handler invocation.
func (h SQSCheckerWrappedHandler) Invoke(ctx context.Context, payload []byte) ([]byte, error) {

	eventType := reflect.TypeOf(events.SQSEvent{})
	event := reflect.New(eventType)
	dec := json.NewDecoder(bytes.NewReader(payload))
	dec.DisallowUnknownFields()
	if err := dec.Decode(event.Interface()); err == nil {
		request := event.Interface().(events.SQSEvent)
		for _, record := range request.Records {
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

// WrapHandler Provides a Handler which wraps customer Handler with OTel Tracing.
func WrapHandlerSQSCheck(handler lambda.Handler) lambda.Handler {
	return SQSCheckerWrappedHandler{handler: handler}
}
