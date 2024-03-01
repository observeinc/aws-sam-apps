package subscriber

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"sync"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/go-logr/logr"
	lambdadetector "go.opentelemetry.io/contrib/detectors/aws/lambda"
	"go.opentelemetry.io/contrib/exporters/autoexport"
	"go.opentelemetry.io/contrib/propagators/b3"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

const (
	instrumentationName    = "github.com/observeinc/aws-sam-apps/cmd/subscriber"
	instrumentationVersion = "0.1.0"
)

var (
	tracerProvider *sdktrace.TracerProvider
	initOnce       sync.Once
	shutdownFn     func(context.Context) error
	tracer         = otel.GetTracerProvider().Tracer(
		instrumentationName,
		trace.WithInstrumentationVersion(instrumentationVersion),
	)
	noopTracer = noop.NewTracerProvider().Tracer(
		instrumentationName,
		trace.WithInstrumentationVersion(instrumentationVersion),
	)
)

type InstrumentedHandler struct {
	*Handler
}

func (h *InstrumentedHandler) HandleSQS(ctx context.Context, request events.SQSEvent) (response events.SQSEventResponse, err error) {
	logger := logr.FromContextOrDiscard(ctx)
	for _, record := range request.Records {
		var req Request
		var err error
		logger.V(3).Info("Getting context from message attributes")
		ctx = NewSQSCarrier().Extract(ctx, record.MessageAttributes)

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

func InstrumentHandler(h *Handler) *InstrumentedHandler {
	ih := InstrumentedHandler{h}
	return &ih
}

type InstrumentedSQSClient struct {
	Client     SQSClient
	Propagator propagation.TextMapPropagator
}

func (q *InstrumentedSQSClient) SendMessage(ctx context.Context, msg *sqs.SendMessageInput, _ ...func(*sqs.Options)) (*sqs.SendMessageOutput, error) {
	logger := logr.FromContextOrDiscard(ctx)
	logger.V(3).Info("Injecting context into message attributes")
	msg.MessageAttributes = make(map[string]types.MessageAttributeValue)
	if err := NewSQSCarrier().Inject(ctx, msg.MessageAttributes); err != nil {
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

func HandleOTELEnvVars() error {
	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	u, err := url.Parse(endpoint)
	if err != nil {
		return fmt.Errorf("failed to parse OTEL_EXPORTER_OTLP_ENDPOINT: %w", err)
	}

	if userinfo := u.User; userinfo != nil {
		authHeader := "Bearer " + userinfo.String()

		headers := os.Getenv("OTEL_EXPORTER_OTLP_HEADERS")
		if headers != "" {
			headers += ","
		}
		headers += "Authorization=" + authHeader

		// remove auth from URL
		u.User = nil
		os.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", u.String())
		os.Setenv("OTEL_EXPORTER_OTLP_HEADERS", headers)
	}
	return nil
}

func InitTracing(ctx context.Context, sn string) (trace.Tracer, func(context.Context) error) {
	if OTLPExporterNoEndpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT") == "" && os.Getenv("OTEL_TRACES_EXPORTER") != "console"; OTLPExporterNoEndpoint {
		return noopTracer, func(_ context.Context) error {
			return nil
		}
	}

	initOnce.Do(func() {
		var err error
		if os.Getenv("OTEL_TRACES_EXPORTER") != "console" {
			err = HandleOTELEnvVars()
			if err != nil {
				shutdownFn = func(_ context.Context) error {
					return fmt.Errorf("failed to parse tracing endpoint: %w", err)
				}
				return
			}
		}

		detector := lambdadetector.NewResourceDetector()
		res, err := resource.New(ctx,
			resource.WithDetectors(detector),
			resource.WithAttributes(semconv.ServiceName(sn)),
		)
		if err != nil {
			shutdownFn = func(_ context.Context) error {
				return fmt.Errorf("failed to create new tracing resource: %w", err)
			}
			return
		}
		exporter, err := autoexport.NewSpanExporter(ctx)
		if err != nil {
			shutdownFn = func(_ context.Context) error {
				return fmt.Errorf("failed to create span exporter: %w", err)
			}
			return
		}
		tracerProvider = sdktrace.NewTracerProvider(
			sdktrace.WithBatcher(exporter),
			sdktrace.WithResource(res),
		)
		shutdownFn = func(ctx context.Context) error {
			if err = tracerProvider.Shutdown(ctx); err != nil {
				return fmt.Errorf("tracer shutdown failed: %w", err)
			}
			return nil
		}
		otel.SetTracerProvider(tracerProvider)
	})
	return tracer, shutdownFn
}
