package tracing

import (
	"context"
	"github.com/observeinc/aws-sam-apps/pkg/handler/subscriber"

	"github.com/aws/aws-lambda-go/lambda"
	"go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-lambda-go/otellambda"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

var payloadKey = attribute.Key("payload")

func NewLambdaHandler(handler lambda.Handler, tp *sdktrace.TracerProvider) lambda.Handler {
	return subscriber.WrapHandlerSQSCheck(otellambda.WrapHandler(
		&LambdaHandler{handler},
		otellambda.WithTracerProvider(tp),
		otellambda.WithFlusher(tp),
	))
}

type LambdaHandler struct {
	lambda.Handler
}

func (h *LambdaHandler) Invoke(ctx context.Context, payload []byte) ([]byte, error) {
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(payloadKey.String(string(payload)))
	resp, err := h.Handler.Invoke(ctx, payload)

	// surprisingly, otellambda wrapper does not emit the error
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}

	// nolint: wrapcheck
	return resp, err
}
