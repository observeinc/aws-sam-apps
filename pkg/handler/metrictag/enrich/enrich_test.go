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

func newEnricher(mock tagging.Client) *Enricher {
	clients := make(map[string]tagging.Client)
	if mock != nil {
		clients["us-east-1"] = mock
	}
	return &Enricher{
		Logger:          discardLogger,
		CacheTTL:        time.Minute,
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

func TestNew_zeroCacheTTL_disablesCaching(t *testing.T) {
	e, err := New(&Config{ResourceCacheTTLSeconds: 0}, discardLogger)
	if err != nil {
		t.Fatal(err)
	}
	if e.CacheTTL != 0 {
		t.Errorf("expected CacheTTL=0 when ResourceCacheTTLSeconds=0, got %v", e.CacheTTL)
	}
}

func TestEnrichLine_unknownNamespace_emptyResourceTags(t *testing.T) {
	e := newEnricher(nil)
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

func TestEnrichLine_allTagsIncluded(t *testing.T) {
	mock := &mockTaggingClient{resources: ec2Resources()}
	e := newEnricher(mock)

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
	e := newEnricher(mock)
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
	e := newEnricher(mock)
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
	e := newEnricher(slow)
	ctx := context.Background()

	const goroutines = 20
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for range goroutines {
		go func() {
			defer wg.Done()
			if _, err := e.getResources(ctx, "123", "us-east-1", "AWS/EC2"); err != nil {
				t.Errorf("getResources error: %v", err)
			}
		}()
	}
	// Hold the underlying call open long enough for every goroutine to
	// enter singleflight.Do for the same key. Without this, the first call
	// can finish (populating the cache) before later goroutines pass the
	// pre-Do cache check, allowing a second underlying invocation through.
	time.Sleep(100 * time.Millisecond)
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
