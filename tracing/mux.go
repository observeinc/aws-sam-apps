package tracing

import (
	"context"

	"github.com/aws/aws-lambda-go/lambda"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type LambdaHandler struct {
	Tracer trace.Tracer
	lambda.Handler
}

func (h *LambdaHandler) Invoke(ctx context.Context, payload []byte) (response []byte, err error) {
	cctx, span := h.Tracer.Start(ctx, "Invoke",
		trace.WithAttributes(attribute.String("payload", string(payload))),
		trace.WithSpanKind(trace.SpanKindServer),
	)
	defer func() {
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()
	response, err = h.Handler.Invoke(cctx, payload)
	return
}
