package tagger

import (
	"context"

	"github.com/aws/aws-lambda-go/events"
	"github.com/go-logr/logr"
)

const (
	formatJSON          = "json"
	formatOpenTelemetry = "opentelemetry"
)

// Handler processes Firehose transformation requests by enriching metric
// records with AWS resource tags.
type Handler struct {
	Cache        *TagCache
	OutputFormat string
	Logger       logr.Logger
}

func New(cfg *Config, cache *TagCache, logger logr.Logger) *Handler {
	format := cfg.OutputFormat
	if format != formatOpenTelemetry {
		format = formatJSON
	}

	return &Handler{
		Cache:        cache,
		OutputFormat: format,
		Logger:       logger,
	}
}

// HandleFirehose processes a Firehose transformation event. Each record is
// enriched with tags and returned with an Ok status. On per-record failure
// the original data is returned with ProcessingFailed status.
func (h *Handler) HandleFirehose(ctx context.Context, event events.KinesisFirehoseEvent) (events.KinesisFirehoseResponse, error) {
	logger := logr.FromContextOrDiscard(ctx)
	logger.V(3).Info("processing firehose event",
		"invocationId", event.InvocationID,
		"recordCount", len(event.Records),
		"format", h.OutputFormat,
	)

	response := events.KinesisFirehoseResponse{
		Records: make([]events.KinesisFirehoseResponseRecord, 0, len(event.Records)),
	}

	for _, record := range event.Records {
		enriched, err := h.enrichRecord(ctx, record.Data)
		if err != nil {
			logger.Error(err, "failed to enrich record", "recordId", record.RecordID)
			response.Records = append(response.Records, events.KinesisFirehoseResponseRecord{
				RecordID: record.RecordID,
				Result:   events.KinesisFirehoseTransformedStateProcessingFailed,
				Data:     record.Data,
			})
			continue
		}

		response.Records = append(response.Records, events.KinesisFirehoseResponseRecord{
			RecordID: record.RecordID,
			Result:   events.KinesisFirehoseTransformedStateOk,
			Data:     enriched,
		})
	}

	return response, nil
}

func (h *Handler) enrichRecord(ctx context.Context, data []byte) ([]byte, error) {
	switch h.OutputFormat {
	case formatOpenTelemetry:
		return enrichOTLP(ctx, data, h.Cache)
	default:
		return enrichJSON(ctx, data, h.Cache)
	}
}

// Warm pre-populates the tag cache. Should be called during Lambda init.
func (h *Handler) Warm(ctx context.Context) {
	h.Cache.Warm(ctx)
}

// WarmHandler wraps Handler to satisfy the Mux registration pattern
// while also warming the cache on first invocation if needed.
func WarmHandler(h *Handler) func(context.Context, events.KinesisFirehoseEvent) (events.KinesisFirehoseResponse, error) {
	return func(ctx context.Context, event events.KinesisFirehoseEvent) (events.KinesisFirehoseResponse, error) {
		return h.HandleFirehose(ctx, event)
	}
}

