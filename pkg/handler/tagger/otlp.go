package tagger

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"

	commonpb "go.opentelemetry.io/proto/otlp/common/v1"
	collectorpb "go.opentelemetry.io/proto/otlp/collector/metrics/v1"
	resourcepb "go.opentelemetry.io/proto/otlp/resource/v1"
	"google.golang.org/protobuf/proto"
)

const namespaceAttrKey = "aws.cloudwatch.namespace"

// enrichOTLP processes a Firehose record containing size-delimited protobuf
// ExportMetricsServiceRequest messages. For each ResourceMetrics, it extracts
// the namespace and dimensions from resource attributes, looks up tags, and
// injects them as additional resource attributes.
func enrichOTLP(ctx context.Context, data []byte, cache *TagCache) ([]byte, error) {
	reader := bytes.NewReader(data)
	var buf bytes.Buffer

	for reader.Len() > 0 {
		var msgLen uint32
		if err := binary.Read(reader, binary.BigEndian, &msgLen); err != nil {
			return nil, fmt.Errorf("read message length: %w", err)
		}

		msgBytes := make([]byte, msgLen)
		if _, err := reader.Read(msgBytes); err != nil {
			return nil, fmt.Errorf("read message body: %w", err)
		}

		var req collectorpb.ExportMetricsServiceRequest
		if err := proto.Unmarshal(msgBytes, &req); err != nil {
			return nil, fmt.Errorf("unmarshal ExportMetricsServiceRequest: %w", err)
		}

		for _, rm := range req.ResourceMetrics {
			if rm.Resource != nil {
				enrichResource(ctx, rm.Resource, cache)
			}
		}

		enrichedBytes, err := proto.Marshal(&req)
		if err != nil {
			return nil, fmt.Errorf("marshal ExportMetricsServiceRequest: %w", err)
		}

		if err := binary.Write(&buf, binary.BigEndian, uint32(len(enrichedBytes))); err != nil {
			return nil, fmt.Errorf("write message length: %w", err)
		}
		buf.Write(enrichedBytes)
	}

	return buf.Bytes(), nil
}

func enrichResource(ctx context.Context, res *resourcepb.Resource, cache *TagCache) {
	if res == nil {
		return
	}

	var namespace string
	dimValues := make(map[string]string)

	for _, attr := range res.Attributes {
		if attr.Key == namespaceAttrKey {
			namespace = attr.Value.GetStringValue()
		} else {
			dimValues[attr.Key] = attr.Value.GetStringValue()
		}
	}

	if namespace == "" {
		return
	}

	mapping, ok := LookupNamespace(namespace)
	if !ok {
		return
	}

	dimValue, exists := dimValues[mapping.DimensionKey]
	if !exists {
		return
	}

	tags := cache.Get(ctx, mapping.ResourceType, dimValue)
	if tags == nil {
		return
	}

	existing := make(map[string]struct{}, len(res.Attributes))
	for _, attr := range res.Attributes {
		existing[attr.Key] = struct{}{}
	}

	for k, v := range tags {
		tagKey := "aws.tag." + k
		if _, ok := existing[tagKey]; ok {
			continue
		}
		res.Attributes = append(res.Attributes, &commonpb.KeyValue{
			Key: tagKey,
			Value: &commonpb.AnyValue{
				Value: &commonpb.AnyValue_StringValue{StringValue: v},
			},
		})
	}
}
