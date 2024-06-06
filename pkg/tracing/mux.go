package tracing

import (
	"context"

	"github.com/aws/aws-lambda-go/lambda"
	"go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-lambda-go/otellambda"
	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

var payloadKey = attribute.Key("payload")

func NewLambdaHandler(handler lambda.Handler, tp *sdktrace.TracerProvider) lambda.Handler {
	return otellambda.WrapHandler(
		&LambdaHandler{handler},
		otellambda.WithTracerProvider(tp),
		otellambda.WithFlusher(tp),
	)
}

type LambdaHandler struct {
	lambda.Handler
}

func (h *LambdaHandler) Invoke(ctx context.Context, payload []byte) ([]byte, error) {
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(payloadKey.String(string(payload)))
	// nolint: wrapcheck
	return h.Handler.Invoke(ctx, payload)
}
