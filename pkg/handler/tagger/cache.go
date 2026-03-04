package tagger

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/go-logr/logr"
)

const cacheFileName = "tagger_cache.json"

// TaggingClient abstracts the Resource Groups Tagging API for testability.
type TaggingClient interface {
	GetResourcesByType(ctx context.Context, resourceType string) (map[string]map[string]string, error)
}

// tagCacheEntry stores a set of tags keyed by resource identifier (dimension
// value) for a single resource type.
type tagCacheEntry struct {
	Tags      map[string]map[string]string `json:"tags"`
	FetchedAt time.Time                    `json:"fetched_at"`
}

// TagCache provides a two-layer cache (in-memory + /tmp file) for resource
// tags. It is safe for concurrent use.
type TagCache struct {
	mu     sync.RWMutex
	client TaggingClient
	ttl    time.Duration
	dir    string
	logger logr.Logger

	// resourceType -> (resourceID -> tags)
	entries map[string]*tagCacheEntry
}

func NewTagCache(client TaggingClient, ttl time.Duration, cacheDir string, logger logr.Logger) *TagCache {
	return &TagCache{
		client:  client,
		ttl:     ttl,
		dir:     cacheDir,
		logger:  logger,
		entries: make(map[string]*tagCacheEntry),
	}
}

// Get returns the tags for a resource identified by its type and dimension
// value. If the cache has expired or is empty, it refreshes from the API
// first. Returns nil (not an error) when no tags are found.
func (c *TagCache) Get(ctx context.Context, resourceType, dimensionValue string) map[string]string {
	c.mu.RLock()
	entry, ok := c.entries[resourceType]
	c.mu.RUnlock()

	if ok && time.Since(entry.FetchedAt) < c.ttl {
		return entry.Tags[dimensionValue]
	}

	if err := c.refreshType(ctx, resourceType); err != nil {
		c.logger.Error(err, "failed to refresh tags", "resourceType", resourceType)
		if ok {
			return entry.Tags[dimensionValue]
		}
		return nil
	}

	c.mu.RLock()
	defer c.mu.RUnlock()
	if e, ok := c.entries[resourceType]; ok {
		return e.Tags[dimensionValue]
	}
	return nil
}

// Warm pre-populates the cache for all known resource types. Called during
// Lambda init to minimize cold-start latency on the first Firehose batch.
func (c *TagCache) Warm(ctx context.Context) {
	if c.loadFromFile() {
		c.logger.V(3).Info("loaded tag cache from file")
		allValid := true
		c.mu.RLock()
		for _, entry := range c.entries {
			if time.Since(entry.FetchedAt) >= c.ttl {
				allValid = false
				break
			}
		}
		c.mu.RUnlock()
		if allValid {
			return
		}
	}

	for _, rt := range AllResourceTypes() {
		if err := c.refreshType(ctx, rt); err != nil {
			c.logger.Error(err, "failed to warm cache", "resourceType", rt)
		}
	}

	c.saveToFile()
}

func (c *TagCache) refreshType(ctx context.Context, resourceType string) error {
	tags, err := c.client.GetResourcesByType(ctx, resourceType)
	if err != nil {
		return fmt.Errorf("get resources for %s: %w", resourceType, err)
	}

	c.mu.Lock()
	c.entries[resourceType] = &tagCacheEntry{
		Tags:      tags,
		FetchedAt: time.Now(),
	}
	c.mu.Unlock()

	c.saveToFile()
	return nil
}

func (c *TagCache) filePath() string {
	return filepath.Join(c.dir, cacheFileName)
}

type fileCacheData struct {
	Entries map[string]*tagCacheEntry `json:"entries"`
}

func (c *TagCache) loadFromFile() bool {
	data, err := os.ReadFile(c.filePath())
	if err != nil {
		return false
	}

	var fc fileCacheData
	if err := json.Unmarshal(data, &fc); err != nil {
		c.logger.Error(err, "failed to unmarshal cache file")
		return false
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries = fc.Entries
	return len(c.entries) > 0
}

func (c *TagCache) saveToFile() {
	c.mu.RLock()
	fc := fileCacheData{Entries: c.entries}
	c.mu.RUnlock()

	data, err := json.Marshal(fc)
	if err != nil {
		c.logger.Error(err, "failed to marshal cache")
		return
	}

	if err := os.WriteFile(c.filePath(), data, 0644); err != nil {
		c.logger.Error(err, "failed to write cache file")
	}
}
