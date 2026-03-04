package tagger

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-logr/logr"
)

type mockTaggingClient struct {
	calls    int
	response map[string]map[string]string
}

func (m *mockTaggingClient) GetResourcesByType(_ context.Context, _ string) (map[string]map[string]string, error) {
	m.calls++
	return m.response, nil
}

func TestTagCache_Get_CacheHit(t *testing.T) {
	t.Parallel()

	client := &mockTaggingClient{
		response: map[string]map[string]string{
			"i-123": {"Name": "web-1", "Env": "prod"},
		},
	}

	dir := t.TempDir()
	cache := NewTagCache(client, 5*time.Minute, dir, logr.Discard())
	ctx := context.Background()

	tags := cache.Get(ctx, "ec2:instance", "i-123")
	if tags == nil {
		t.Fatal("expected tags, got nil")
	}
	if tags["Name"] != "web-1" {
		t.Errorf("Name=%q, want %q", tags["Name"], "web-1")
	}
	if client.calls != 1 {
		t.Errorf("calls=%d, want 1", client.calls)
	}

	tags = cache.Get(ctx, "ec2:instance", "i-123")
	if tags == nil {
		t.Fatal("expected cached tags, got nil")
	}
	if client.calls != 1 {
		t.Errorf("calls=%d after cache hit, want 1", client.calls)
	}
}

func TestTagCache_Get_CacheMiss(t *testing.T) {
	t.Parallel()

	client := &mockTaggingClient{
		response: map[string]map[string]string{
			"i-123": {"Name": "web-1"},
		},
	}

	dir := t.TempDir()
	cache := NewTagCache(client, 5*time.Minute, dir, logr.Discard())
	ctx := context.Background()

	tags := cache.Get(ctx, "ec2:instance", "i-999")
	if tags != nil {
		t.Errorf("expected nil for missing resource, got %v", tags)
	}
}

func TestTagCache_Get_Expiry(t *testing.T) {
	t.Parallel()

	client := &mockTaggingClient{
		response: map[string]map[string]string{
			"i-123": {"Name": "web-1"},
		},
	}

	dir := t.TempDir()
	cache := NewTagCache(client, 1*time.Millisecond, dir, logr.Discard())
	ctx := context.Background()

	cache.Get(ctx, "ec2:instance", "i-123")
	if client.calls != 1 {
		t.Fatalf("calls=%d, want 1", client.calls)
	}

	time.Sleep(5 * time.Millisecond)

	cache.Get(ctx, "ec2:instance", "i-123")
	if client.calls != 2 {
		t.Errorf("calls=%d after expiry, want 2", client.calls)
	}
}

func TestTagCache_FileCache(t *testing.T) {
	t.Parallel()

	client := &mockTaggingClient{
		response: map[string]map[string]string{
			"i-123": {"Name": "web-1"},
		},
	}

	dir := t.TempDir()
	cache := NewTagCache(client, 5*time.Minute, dir, logr.Discard())
	ctx := context.Background()

	cache.Get(ctx, "ec2:instance", "i-123")

	fp := filepath.Join(dir, cacheFileName)
	if _, err := os.Stat(fp); os.IsNotExist(err) {
		t.Fatal("cache file was not created")
	}

	cache2 := NewTagCache(client, 5*time.Minute, dir, logr.Discard())
	cache2.Warm(ctx)

	tags := cache2.Get(ctx, "ec2:instance", "i-123")
	if tags == nil {
		t.Fatal("expected tags loaded from file cache")
	}
	if tags["Name"] != "web-1" {
		t.Errorf("Name=%q, want %q", tags["Name"], "web-1")
	}
}
