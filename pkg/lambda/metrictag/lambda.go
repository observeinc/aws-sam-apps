package metrictag

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"

	"github.com/aws/aws-lambda-go/events"
	"github.com/go-logr/logr"

	"github.com/observeinc/aws-sam-apps/pkg/handler/metrictag/enrich"
	"github.com/observeinc/aws-sam-apps/pkg/logging"
)

// Config is loaded from the Lambda environment (see lambda.ProcessEnv).
type Config struct {
	ResourceCacheTTLSeconds int    `env:"RESOURCE_CACHE_TTL_SECONDS"`
	ResourceTagKeys         string `env:"RESOURCE_TAG_KEYS"`
	YACESearchTags          string `env:"YACE_SEARCH_TAGS"`
	AssumeRoleARN           string `env:"ASSUME_ROLE_ARN"`

	Logging *logging.Config
}

// Lambda wires Kinesis Firehose transform + tag enrichment.
type Lambda struct {
	Logger     logr.Logger
	Entrypoint func(context.Context, events.KinesisFirehoseEvent) (events.KinesisFirehoseResponse, error)
	Shutdown   func()

	enricher *enrich.Enricher
}

// New builds the Firehose transform handler.
func New(ctx context.Context, cfg *Config) (*Lambda, error) {
	logger := logging.New(cfg.Logging)
	logger.V(4).Info("initialized", "config", cfg)

	slogLog := slog.New(logr.ToSlogHandler(logger))
	e, err := enrich.New(&enrich.Config{
		ResourceCacheTTLSeconds: cfg.ResourceCacheTTLSeconds,
		ResourceTagKeysCSV:      cfg.ResourceTagKeys,
		YACESearchTags:          cfg.YACESearchTags,
		AssumeRoleARN:           cfg.AssumeRoleARN,
	}, slogLog)
	if err != nil {
		return nil, fmt.Errorf("enricher: %w", err)
	}

	l := &Lambda{
		Logger:   logger,
		enricher: e,
		Shutdown: func() {
			logger.V(4).Info("SIGTERM received, running shutdown")
		},
	}
	l.Entrypoint = l.handle
	return l, nil
}

func (l *Lambda) handle(ctx context.Context, ev events.KinesisFirehoseEvent) (events.KinesisFirehoseResponse, error) {
	var out events.KinesisFirehoseResponse
	out.Records = make([]events.KinesisFirehoseResponseRecord, 0, len(ev.Records))

	for _, rec := range ev.Records {
		raw := rec.Data

		// > each Firehose record contains multiple JSON objects separated by a newline character
		// https://docs.aws.amazon.com/AmazonCloudWatch/latest/monitoring/CloudWatch-metric-streams-formats-json.html
		lines := bytes.Split(bytes.TrimSpace(raw), []byte("\n"))
		var buf bytes.Buffer
		ok := true
		for i, line := range lines {
			line = bytes.TrimSpace(line)
			if len(line) == 0 {
				continue
			}
			enriched, err := l.enricher.EnrichLine(ctx, line)
			if err != nil {
				l.Logger.Info("line enrich failed", "recordId", rec.RecordID, "line", i, "err", err)
				ok = false
				break
			}
			if buf.Len() > 0 {
				buf.WriteByte('\n')
			}
			buf.Write(enriched)
		}

		if !ok {
			out.Records = append(out.Records, events.KinesisFirehoseResponseRecord{
				RecordID: rec.RecordID,
				Result:   events.KinesisFirehoseTransformedStateProcessingFailed,
				Data:     rec.Data,
			})
			continue
		}

		if buf.Len() > 0 {
			buf.WriteByte('\n')
		}
		out.Records = append(out.Records, events.KinesisFirehoseResponseRecord{
			RecordID: rec.RecordID,
			Result:   events.KinesisFirehoseTransformedStateOk,
			Data:     buf.Bytes(),
		})
	}

	return out, nil
}
