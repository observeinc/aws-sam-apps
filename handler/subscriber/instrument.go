package subscriber

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
	"github.com/go-logr/logr"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

type InstrumentedHandler struct {
	Handler
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
	fmt.Println("Running instrumented handle request")
	if req.TraceContext != nil {
		fmt.Println("Found trace context, extracting")
		propagator := propagation.NewCompositeTextMapPropagator(propagation.Baggage{}, propagation.TraceContext{})
		ctx = propagator.Extract(ctx, req.TraceContext)
	}
	ctx, span := h.Tracer.Start(ctx, "HandleRequest", trace.WithAttributes(attribute.String("key1", "value1")))
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

func InstrumentHandler(h *Handler) *InstrumentedHandler {
	ih := InstrumentedHandler{*h}
	instrumentedQueue := InstrumentQueue(h.Queue)
	ih.Queue = instrumentedQueue
	return &ih
}

func New(cfg *Config) (*Handler, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	h := &Handler{
		Client:     cfg.CloudWatchLogsClient,
		Queue:      cfg.Queue,
		NumWorkers: cfg.NumWorkers,
		subscriptionFilter: types.SubscriptionFilter{
			FilterName:     aws.String(cfg.FilterName),
			FilterPattern:  aws.String(cfg.FilterPattern),
			DestinationArn: aws.String(cfg.DestinationARN),
			RoleArn:        cfg.RoleARN,
		},
		logGroupNameFilter: cfg.LogGroupFilter(),
	}

	if h.NumWorkers <= 0 {
		h.NumWorkers = runtime.NumCPU()
	}

	if cfg.Logger != nil {
		h.Logger = *cfg.Logger
	}

	if cfg.Tracer != nil {
		h.Tracer = cfg.Tracer
	}

	return h, nil
}

type instrumentedQueueWrapper struct {
	queue      Queue
	Propagator propagation.TextMapPropagator
}

func (q *instrumentedQueueWrapper) Put(ctx context.Context, items ...*Request) error {
	for _, item := range items {
		carrier := propagation.MapCarrier{}
		q.Propagator.Inject(ctx, carrier)
		item.TraceContext = &carrier
	}
	return q.queue.Put(ctx, items...)
}

func InstrumentQueue(q Queue) Queue {
	propagator := propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{})
	instrumentedQueue := &instrumentedQueueWrapper{
		queue:      q,
		Propagator: propagator,
	}
	return instrumentedQueue
}
