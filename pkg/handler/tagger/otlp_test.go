package tagger

import (
	"bytes"
	"context"
	"encoding/binary"
	"testing"
	"time"

	"github.com/go-logr/logr"
	commonpb "go.opentelemetry.io/proto/otlp/common/v1"
	collectorpb "go.opentelemetry.io/proto/otlp/collector/metrics/v1"
	metricspb "go.opentelemetry.io/proto/otlp/metrics/v1"
	resourcepb "go.opentelemetry.io/proto/otlp/resource/v1"
	"google.golang.org/protobuf/proto"
)

func makeSizeDelimited(t *testing.T, req *collectorpb.ExportMetricsServiceRequest) []byte {
	t.Helper()
	data, err := proto.Marshal(req)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var buf bytes.Buffer
	binary.Write(&buf, binary.BigEndian, uint32(len(data)))
	buf.Write(data)
	return buf.Bytes()
}

func TestEnrichOTLP(t *testing.T) {
	t.Parallel()

	client := &mockTaggingClient{
		response: map[string]map[string]string{
			"i-0abcd1234": {"Name": "web-server-1", "Team": "platform"},
		},
	}

	cache := NewTagCache(client, 5*time.Minute, t.TempDir(), logr.Discard())
	ctx := context.Background()

	req := &collectorpb.ExportMetricsServiceRequest{
		ResourceMetrics: []*metricspb.ResourceMetrics{
			{
				Resource: &resourcepb.Resource{
					Attributes: []*commonpb.KeyValue{
						{
							Key:   namespaceAttrKey,
							Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: "AWS/EC2"}},
						},
						{
							Key:   "InstanceId",
							Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: "i-0abcd1234"}},
						},
					},
				},
				ScopeMetrics: []*metricspb.ScopeMetrics{
					{
						Metrics: []*metricspb.Metric{
							{Name: "CPUUtilization"},
						},
					},
				},
			},
		},
	}

	input := makeSizeDelimited(t, req)
	result, err := enrichOTLP(ctx, input, cache)
	if err != nil {
		t.Fatalf("enrichOTLP error: %v", err)
	}

	reader := bytes.NewReader(result)
	var msgLen uint32
	if err := binary.Read(reader, binary.BigEndian, &msgLen); err != nil {
		t.Fatalf("read length: %v", err)
	}

	msgBytes := make([]byte, msgLen)
	reader.Read(msgBytes)

	var enrichedReq collectorpb.ExportMetricsServiceRequest
	if err := proto.Unmarshal(msgBytes, &enrichedReq); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	attrs := enrichedReq.ResourceMetrics[0].Resource.Attributes
	tagAttrs := make(map[string]string)
	for _, a := range attrs {
		if len(a.Key) > 8 && a.Key[:8] == "aws.tag." {
			tagAttrs[a.Key[8:]] = a.Value.GetStringValue()
		}
	}

	if tagAttrs["Name"] != "web-server-1" {
		t.Errorf("tag Name=%q, want %q", tagAttrs["Name"], "web-server-1")
	}
	if tagAttrs["Team"] != "platform" {
		t.Errorf("tag Team=%q, want %q", tagAttrs["Team"], "platform")
	}
}

func TestEnrichOTLP_UnknownNamespace(t *testing.T) {
	t.Parallel()

	client := &mockTaggingClient{
		response: map[string]map[string]string{},
	}

	cache := NewTagCache(client, 5*time.Minute, t.TempDir(), logr.Discard())
	ctx := context.Background()

	req := &collectorpb.ExportMetricsServiceRequest{
		ResourceMetrics: []*metricspb.ResourceMetrics{
			{
				Resource: &resourcepb.Resource{
					Attributes: []*commonpb.KeyValue{
						{
							Key:   namespaceAttrKey,
							Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: "Custom/MyApp"}},
						},
					},
				},
			},
		},
	}

	input := makeSizeDelimited(t, req)
	result, err := enrichOTLP(ctx, input, cache)
	if err != nil {
		t.Fatalf("enrichOTLP error: %v", err)
	}

	reader := bytes.NewReader(result)
	var msgLen uint32
	binary.Read(reader, binary.BigEndian, &msgLen)
	msgBytes := make([]byte, msgLen)
	reader.Read(msgBytes)

	var enrichedReq collectorpb.ExportMetricsServiceRequest
	proto.Unmarshal(msgBytes, &enrichedReq)

	attrs := enrichedReq.ResourceMetrics[0].Resource.Attributes
	for _, a := range attrs {
		if len(a.Key) > 8 && a.Key[:8] == "aws.tag." {
			t.Errorf("unexpected tag attribute: %s", a.Key)
		}
	}
}

func TestEnrichResource_NilResource(t *testing.T) {
	t.Parallel()

	client := &mockTaggingClient{
		response: map[string]map[string]string{},
	}
	cache := NewTagCache(client, 5*time.Minute, t.TempDir(), logr.Discard())
	ctx := context.Background()

	enrichResource(ctx, nil, cache)
}
