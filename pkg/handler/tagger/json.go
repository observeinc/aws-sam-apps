package tagger

import (
	"bytes"
	"context"
	"encoding/json"
)

// cloudwatchMetricJSON represents the JSON format emitted by CloudWatch
// Metric Streams.
type cloudwatchMetricJSON struct {
	MetricStreamName string                 `json:"metric_stream_name"`
	AccountID        string                 `json:"account_id"`
	Region           string                 `json:"region"`
	Namespace        string                 `json:"namespace"`
	MetricName       string                 `json:"metric_name"`
	Dimensions       map[string]string      `json:"dimensions"`
	Timestamp        int64                  `json:"timestamp"`
	Value            map[string]interface{} `json:"value"`
	Unit             string                 `json:"unit"`
	Tags             map[string]string      `json:"tags,omitempty"`
}

// enrichJSON processes a Firehose record containing newline-delimited JSON
// CloudWatch metric data, looks up tags for each metric, and injects them
// into the record. Returns the enriched record bytes.
func enrichJSON(ctx context.Context, data []byte, cache *TagCache) ([]byte, error) {
	lines := bytes.Split(data, []byte("\n"))
	var buf bytes.Buffer

	for _, line := range lines {
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}

		var metric cloudwatchMetricJSON
		if err := json.Unmarshal(line, &metric); err != nil {
			buf.Write(line)
			buf.WriteByte('\n')
			continue
		}

		mapping, ok := LookupNamespace(metric.Namespace)
		if ok {
			if dimValue, exists := metric.Dimensions[mapping.DimensionKey]; exists {
				if tags := cache.Get(ctx, mapping.ResourceType, dimValue); tags != nil {
					metric.Tags = tags
				}
			}
		}

		enriched, err := json.Marshal(metric)
		if err != nil {
			buf.Write(line)
			buf.WriteByte('\n')
			continue
		}

		buf.Write(enriched)
		buf.WriteByte('\n')
	}

	return buf.Bytes(), nil
}
