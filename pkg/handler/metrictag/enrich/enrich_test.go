package enrich

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/prometheus-community/yet-another-cloudwatch-exporter/pkg/clients/tagging"
	"github.com/prometheus-community/yet-another-cloudwatch-exporter/pkg/model"
)

var discardLogger = slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))

type mockTaggingClient struct {
	resources []*model.TaggedResource
	err       error
	callCount atomic.Int32
}

func (m *mockTaggingClient) GetResources(_ context.Context, _ model.DiscoveryJob, _ string) ([]*model.TaggedResource, error) {
	m.callCount.Add(1)
	return m.resources, m.err
}

func newEnricher(mock tagging.Client, tagKeys map[string]struct{}) *Enricher {
	clients := make(map[string]tagging.Client)
	if mock != nil {
		clients["us-east-1"] = mock
	}
	return &Enricher{
		Logger:          discardLogger,
		CacheTTL:        time.Minute,
		TagKeys:         tagKeys,
		cache:           make(map[string]*cacheEntry),
		taggingByRegion: clients,
	}
}

func ec2Resources() []*model.TaggedResource {
	return []*model.TaggedResource{
		{
			ARN:       "arn:aws:ec2:us-east-1:123:instance/i-abc123",
			Namespace: "AWS/EC2",
			Tags: []model.Tag{
				{Key: "Environment", Value: "prod"},
				{Key: "Team", Value: "platform"},
			},
		},
	}
}

const ec2Line = `{"namespace":"AWS/EC2","region":"us-east-1","account_id":"123","metric_name":"CPUUtilization","dimensions":{"InstanceId":"i-abc123"},"timestamp":1,"value":{"count":1},"unit":"Count"}`

func TestParseSearchTagsEnv(t *testing.T) {
	tags, err := parseSearchTagsEnv("Environment=^prod$,Team=.*")
	if err != nil {
		t.Fatal(err)
	}
	if len(tags) != 2 || tags[0].Key != "Environment" || tags[1].Key != "Team" {
		t.Fatalf("unexpected tags: %+v", tags)
	}
}

func TestEnrichLine_unknownNamespace_emptyResourceTags(t *testing.T) {
	e := newEnricher(nil, nil)
	line := `{"namespace":"Custom/Unknown","region":"us-east-1","account_id":"123","metric_name":"M","dimensions":{},"timestamp":1,"value":{"count":1},"unit":"Count"}`
	out, err := e.EnrichLine(context.Background(), []byte(line))
	if err != nil {
		t.Fatal(err)
	}
	var obj map[string]any
	if err := json.Unmarshal(out, &obj); err != nil {
		t.Fatal(err)
	}
	rt, ok := obj["resource_tags"].(map[string]any)
	if !ok {
		t.Fatalf("resource_tags missing or wrong type: %#v", obj["resource_tags"])
	}
	if len(rt) != 0 {
		t.Fatalf("expected empty resource_tags, got %#v", rt)
	}
}

func TestEnrichLine_tagKeyFiltering(t *testing.T) {
	mock := &mockTaggingClient{resources: ec2Resources()}
	e := newEnricher(mock, map[string]struct{}{"Environment": {}})

	out, err := e.EnrichLine(context.Background(), []byte(ec2Line))
	if err != nil {
		t.Fatal(err)
	}
	var obj map[string]any
	if err := json.Unmarshal(out, &obj); err != nil {
		t.Fatal(err)
	}
	rt, ok := obj["resource_tags"].(map[string]any)
	if !ok {
		t.Fatalf("resource_tags missing or wrong type: %#v", obj["resource_tags"])
	}
	if _, has := rt["Environment"]; !has {
		t.Errorf("expected Environment tag, got %v", rt)
	}
	if _, has := rt["Team"]; has {
		t.Errorf("Team tag should have been filtered out, got %v", rt)
	}
}

func TestEnrichLine_noTagKeyFilter_allTagsIncluded(t *testing.T) {
	mock := &mockTaggingClient{resources: ec2Resources()}
	e := newEnricher(mock, nil)

	out, err := e.EnrichLine(context.Background(), []byte(ec2Line))
	if err != nil {
		t.Fatal(err)
	}
	var obj map[string]any
	if err := json.Unmarshal(out, &obj); err != nil {
		t.Fatal(err)
	}
	rt, ok := obj["resource_tags"].(map[string]any)
	if !ok {
		t.Fatalf("resource_tags missing or wrong type: %#v", obj["resource_tags"])
	}
	if rt["Environment"] != "prod" || rt["Team"] != "platform" {
		t.Errorf("expected both tags, got %v", rt)
	}
}

func TestGetResources_cachePreventsDuplicateCalls(t *testing.T) {
	mock := &mockTaggingClient{resources: ec2Resources()}
	e := newEnricher(mock, nil)
	ctx := context.Background()

	if _, err := e.getResources(ctx, "123", "us-east-1", "AWS/EC2"); err != nil {
		t.Fatal(err)
	}
	if _, err := e.getResources(ctx, "123", "us-east-1", "AWS/EC2"); err != nil {
		t.Fatal(err)
	}

	if n := mock.callCount.Load(); n != 1 {
		t.Errorf("expected 1 GetResources call, got %d", n)
	}
}

func TestGetResources_expiredCacheRefetches(t *testing.T) {
	mock := &mockTaggingClient{resources: ec2Resources()}
	e := newEnricher(mock, nil)
	e.CacheTTL = -time.Second
	ctx := context.Background()

	if _, err := e.getResources(ctx, "123", "us-east-1", "AWS/EC2"); err != nil {
		t.Fatal(err)
	}
	if _, err := e.getResources(ctx, "123", "us-east-1", "AWS/EC2"); err != nil {
		t.Fatal(err)
	}

	if n := mock.callCount.Load(); n != 2 {
		t.Errorf("expected 2 GetResources calls after expiry, got %d", n)
	}
}

func TestGetResources_concurrentCallsDeduplicated(t *testing.T) {
	ready := make(chan struct{})
	mock := &mockTaggingClient{}
	mock.resources = ec2Resources()

	slow := &slowTaggingClient{inner: mock, ready: ready}
	e := newEnricher(slow, nil)
	ctx := context.Background()

	const goroutines = 20
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for range goroutines {
		go func() {
			defer wg.Done()
			<-ready
			if _, err := e.getResources(ctx, "123", "us-east-1", "AWS/EC2"); err != nil {
				t.Errorf("getResources error: %v", err)
			}
		}()
	}
	close(ready)
	wg.Wait()

	if n := mock.callCount.Load(); n != 1 {
		t.Errorf("singleflight should deduplicate to 1 call, got %d", n)
	}
}

type slowTaggingClient struct {
	inner tagging.Client
	ready chan struct{}
}

func (s *slowTaggingClient) GetResources(ctx context.Context, job model.DiscoveryJob, region string) ([]*model.TaggedResource, error) {
	<-s.ready
	return s.inner.GetResources(ctx, job, region)
}
