// Package enrich attaches AWS resource tags to CloudWatch metric stream JSON lines using YACE discovery primitives.
package enrich

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/amp"
	"github.com/aws/aws-sdk-go-v2/service/apigateway"
	"github.com/aws/aws-sdk-go-v2/service/apigatewayv2"
	"github.com/aws/aws-sdk-go-v2/service/autoscaling"
	"github.com/aws/aws-sdk-go-v2/service/databasemigrationservice"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/resourcegroupstaggingapi"
	"github.com/aws/aws-sdk-go-v2/service/shield"
	"github.com/aws/aws-sdk-go-v2/service/storagegateway"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/grafana/regexp"
	"github.com/prometheus-community/yet-another-cloudwatch-exporter/pkg/clients/tagging"
	taggingv2 "github.com/prometheus-community/yet-another-cloudwatch-exporter/pkg/clients/tagging/v2"
	"github.com/prometheus-community/yet-another-cloudwatch-exporter/pkg/config"
	"github.com/prometheus-community/yet-another-cloudwatch-exporter/pkg/job/maxdimassociator"
	"github.com/prometheus-community/yet-another-cloudwatch-exporter/pkg/model"
	"golang.org/x/sync/singleflight"
)

// Config holds enricher options (from environment via pkg/lambda metrictag.Config).
type Config struct {
	ResourceCacheTTLSeconds int
	ResourceTagKeysCSV      string
	YACESearchTags          string
	// AssumeRoleARN, if non-empty, is an IAM role the enricher assumes via STS
	// before calling tagging/discovery APIs. Enables cross-account enrichment.
	AssumeRoleARN string
}

// Enricher adds resource_tags to metric stream JSON objects (one per line).
type Enricher struct {
	Logger *slog.Logger

	// TagKeys, if non-empty, restricts which tag keys are copied into resource_tags.
	TagKeys map[string]struct{}
	// SearchTags filters discovered resources (YACE semantics).
	SearchTags []model.SearchTag
	// CacheTTL is how long tagged resource lists are reused per cache key.
	CacheTTL time.Duration
	// AssumeRoleARN, if non-empty, is assumed via STS before AWS calls.
	AssumeRoleARN string

	mu              sync.Mutex
	sf              singleflight.Group
	taggingByRegion map[string]tagging.Client
	cache           map[string]*cacheEntry
}

type cacheEntry struct {
	assoc   maxdimassociator.Associator
	expires time.Time
}

// New builds an Enricher from config. A nil cfg uses the same defaults as the former env-only bootstrap.
func New(cfg *Config, logger *slog.Logger) (*Enricher, error) {
	if logger == nil {
		logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	}
	if cfg == nil {
		cfg = &Config{}
	}
	ttl := 10 * time.Minute
	if cfg.ResourceCacheTTLSeconds > 0 {
		ttl = time.Duration(cfg.ResourceCacheTTLSeconds) * time.Second
	}
	var tagKeys map[string]struct{}
	if keys := strings.TrimSpace(cfg.ResourceTagKeysCSV); keys != "" {
		tagKeys = make(map[string]struct{})
		for _, k := range strings.Split(keys, ",") {
			k = strings.TrimSpace(k)
			if k != "" {
				tagKeys[k] = struct{}{}
			}
		}
	}
	searchTags, err := parseSearchTagsEnv(cfg.YACESearchTags)
	if err != nil {
		return nil, err
	}
	return &Enricher{
		Logger:          logger,
		TagKeys:         tagKeys,
		SearchTags:      searchTags,
		CacheTTL:        ttl,
		AssumeRoleARN:   strings.TrimSpace(cfg.AssumeRoleARN),
		taggingByRegion: make(map[string]tagging.Client),
		cache:           make(map[string]*cacheEntry),
	}, nil
}

// parseSearchTagsEnv parses a comma-separated list of "Key=regex" pairs from
// the YACE_SEARCH_TAGS environment variable. An empty or absent value for a key
// defaults to matching everything (".*").
func parseSearchTagsEnv(s string) ([]model.SearchTag, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	var out []model.SearchTag
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		key, pattern, ok := strings.Cut(part, "=")
		key = strings.TrimSpace(key)
		if key == "" {
			return nil, fmt.Errorf("invalid YACE_SEARCH_TAGS segment %q", part)
		}
		if !ok {
			pattern = ".*"
		} else {
			pattern = strings.TrimSpace(pattern)
			if pattern == "" {
				pattern = ".*"
			}
		}
		re, err := regexp.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf("YACE_SEARCH_TAGS regex for key %q: %w", key, err)
		}
		out = append(out, model.SearchTag{Key: key, Value: re})
	}
	return out, nil
}

// taggingClient returns a cached YACE tagging client for the given region,
// creating one if it does not yet exist. The mutex is released before the
// network call to avoid serializing concurrent goroutines on a cold region.
func (e *Enricher) taggingClient(ctx context.Context, region string) (tagging.Client, error) {
	e.mu.Lock()
	if c, ok := e.taggingByRegion[region]; ok {
		e.mu.Unlock()
		return c, nil
	}
	e.mu.Unlock()

	cfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(region))
	if err != nil {
		return nil, err
	}
	if e.AssumeRoleARN != "" {
		stsClient := sts.NewFromConfig(cfg)
		provider := stscreds.NewAssumeRoleProvider(stsClient, e.AssumeRoleARN, func(o *stscreds.AssumeRoleOptions) {
			o.RoleSessionName = "metrictag-enrich"
		})
		cfg.Credentials = aws.NewCredentialsCache(provider)
	}
	c := taggingv2.NewClient(
		e.Logger,
		resourcegroupstaggingapi.NewFromConfig(cfg),
		autoscaling.NewFromConfig(cfg),
		apigateway.NewFromConfig(cfg),
		apigatewayv2.NewFromConfig(cfg),
		ec2.NewFromConfig(cfg),
		databasemigrationservice.NewFromConfig(cfg),
		amp.NewFromConfig(cfg),
		storagegateway.NewFromConfig(cfg),
		shield.NewFromConfig(cfg),
	)

	e.mu.Lock()
	if existing, ok := e.taggingByRegion[region]; ok {
		e.mu.Unlock()
		return existing, nil
	}
	e.taggingByRegion[region] = c
	e.mu.Unlock()
	return c, nil
}

func cacheKey(accountID, region, namespace string) string {
	return accountID + "\x00" + region + "\x00" + namespace
}

// getResources returns a cached cacheEntry (associator + expiry) for the given
// account/region/namespace, fetching and building it via the tagging API if
// the cache is cold or expired. Returns nil if the namespace is unsupported.
func (e *Enricher) getResources(ctx context.Context, accountID, region, namespace string) (*cacheEntry, error) {
	svc := config.SupportedServices.GetService(namespace)
	if svc == nil {
		return nil, nil
	}

	key := cacheKey(accountID, region, namespace)

	e.mu.Lock()
	if ent, ok := e.cache[key]; ok && time.Now().Before(ent.expires) {
		e.mu.Unlock()
		return ent, nil
	}
	e.mu.Unlock()

	v, err, _ := e.sf.Do(key, func() (any, error) {
		client, err := e.taggingClient(ctx, region)
		if err != nil {
			return nil, err
		}

		job := model.DiscoveryJob{
			Namespace:  namespace,
			SearchTags: e.SearchTags,
			Regions:    []string{region},
			Metrics:    nil,
			CustomTags: nil,
		}

		resources, err := client.GetResources(ctx, job, region)
		if err != nil {
			if errors.Is(err, tagging.ErrExpectedToFindResources) {
				e.Logger.Debug("no tagged resources for namespace", "namespace", namespace, "region", region)
				resources = nil
			} else {
				return nil, err
			}
		}

		entry := &cacheEntry{
			assoc:   maxdimassociator.NewAssociator(e.Logger, svc.ToModelDimensionsRegexp(), resources),
			expires: time.Now().Add(e.CacheTTL),
		}
		e.mu.Lock()
		e.cache[key] = entry
		e.mu.Unlock()

		return entry, nil
	})
	if err != nil {
		return nil, err
	}
	if v == nil {
		return nil, nil
	}
	return v.(*cacheEntry), nil
}

// EnrichLine parses one JSON object, adds resource_tags, and re-marshals JSON.
// Always preserves the metric: on association failure resource_tags is {}.
func (e *Enricher) EnrichLine(ctx context.Context, lineJSON []byte) ([]byte, error) {
	var obj map[string]any
	if err := json.Unmarshal(lineJSON, &obj); err != nil {
		return nil, err
	}

	ns, _ := obj["namespace"].(string)
	region, _ := obj["region"].(string)
	accountID, _ := obj["account_id"].(string)
	metricName, _ := obj["metric_name"].(string)

	dimensions := map[string]string{}
	if raw, ok := obj["dimensions"].(map[string]any); ok {
		for k, v := range raw {
			if s, ok := v.(string); ok {
				dimensions[k] = s
			}
		}
	}

	tags := e.tagsForMetric(ctx, accountID, region, ns, metricName, dimensions)
	if tags == nil {
		tags = map[string]string{}
	}
	obj["resource_tags"] = tags

	out, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// tagsForMetric resolves the AWS resource tags for a single CloudWatch metric.
// Returns an empty map (never nil) when no matching resource is found.
func (e *Enricher) tagsForMetric(ctx context.Context, accountID, region, namespace, metricName string, dimensions map[string]string) map[string]string {
	entry, err := e.getResources(ctx, accountID, region, namespace)
	if err != nil {
		e.Logger.Warn("getResources failed", "err", err, "namespace", namespace, "region", region)
		return map[string]string{}
	}
	if entry == nil {
		return map[string]string{}
	}

	// Dimension order does not matter; the associator fingerprints via a map.
	dims := make([]model.Dimension, 0, len(dimensions))
	for k, v := range dimensions {
		dims = append(dims, model.Dimension{Name: k, Value: v})
	}

	cw := &model.Metric{
		Namespace:  namespace,
		MetricName: metricName,
		Dimensions: dims,
	}

	res, _ := entry.assoc.AssociateMetricToResource(cw)
	if res == nil {
		return map[string]string{}
	}

	out := make(map[string]string)
	for _, t := range res.Tags {
		if len(e.TagKeys) > 0 {
			if _, ok := e.TagKeys[t.Key]; !ok {
				continue
			}
		}
		out[t.Key] = t.Value
	}
	return out
}
