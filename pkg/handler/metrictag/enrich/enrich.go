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

	"github.com/aws/aws-lambda-go/lambdacontext"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/amp"
	"github.com/aws/aws-sdk-go-v2/service/apigateway"
	"github.com/aws/aws-sdk-go-v2/service/apigatewayv2"
	"github.com/aws/aws-sdk-go-v2/service/autoscaling"
	"github.com/aws/aws-sdk-go-v2/service/databasemigrationservice"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/resourcegroupstaggingapi"
	"github.com/aws/aws-sdk-go-v2/service/shield"
	"github.com/aws/aws-sdk-go-v2/service/storagegateway"
	"github.com/prometheus-community/yet-another-cloudwatch-exporter/pkg/clients/tagging"
	"github.com/prometheus-community/yet-another-cloudwatch-exporter/pkg/config"
	"github.com/prometheus-community/yet-another-cloudwatch-exporter/pkg/job/maxdimassociator"
	"github.com/prometheus-community/yet-another-cloudwatch-exporter/pkg/model"
	"golang.org/x/sync/singleflight"
)

// Config holds enricher options (from environment via pkg/lambda metrictag.Config).
type Config struct {
	ResourceCacheTTLSeconds int
}

const (
	aliasCacheTTL      = time.Hour
	aliasErrorCacheTTL = 5 * time.Minute
)

// AccountAliasFunc returns the account ID the Lambda runs in and its IAM
// account alias (empty if none is set).
type AccountAliasFunc func(ctx context.Context) (accountID, alias string, err error)

// Enricher adds resource_tags to metric stream JSON objects (one per line).
type Enricher struct {
	Logger *slog.Logger

	// CacheTTL is how long tagged resource lists are reused per cache key.
	CacheTTL time.Duration

	// AccountAlias resolves the local account alias. IAM aliases can only be
	// read for the caller's own account, so records from other (e.g. OAM
	// source) accounts are not annotated.
	AccountAlias AccountAliasFunc

	aliasMu      sync.Mutex
	aliasAccount string
	aliasValue   string
	aliasExpires time.Time

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
	// ttl=0 disables caching: cache entries expire at time.Now(), so the
	// strict time.Now().Before(expires) check in getResources is always false.
	ttl := time.Duration(cfg.ResourceCacheTTLSeconds) * time.Second
	return &Enricher{
		Logger:          logger,
		CacheTTL:        ttl,
		AccountAlias:    lookupAccountAlias,
		taggingByRegion: make(map[string]tagging.Client),
		cache:           make(map[string]*cacheEntry),
	}, nil
}

// lookupAccountAlias reads the local account ID from the invoked function ARN
// (arn:aws:lambda:region:account:function:name) and its first IAM alias.
func lookupAccountAlias(ctx context.Context) (string, string, error) {
	lc, ok := lambdacontext.FromContext(ctx)
	if !ok {
		return "", "", errors.New("no lambda context")
	}
	parts := strings.Split(lc.InvokedFunctionArn, ":")
	if len(parts) < 5 || parts[4] == "" {
		return "", "", fmt.Errorf("unexpected function ARN %q", lc.InvokedFunctionArn)
	}
	accountID := parts[4]

	cfg, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		return accountID, "", err
	}
	out, err := iam.NewFromConfig(cfg).ListAccountAliases(ctx, &iam.ListAccountAliasesInput{})
	if err != nil {
		return accountID, "", err
	}
	if len(out.AccountAliases) > 0 {
		return accountID, out.AccountAliases[0], nil
	}
	return accountID, "", nil
}

// accountAlias returns the cached alias for accountID, or "" if accountID is
// not the local account or no alias is configured.
func (e *Enricher) accountAlias(ctx context.Context, accountID string) string {
	if e.AccountAlias == nil || accountID == "" {
		return ""
	}
	e.aliasMu.Lock()
	defer e.aliasMu.Unlock()
	if time.Now().After(e.aliasExpires) {
		id, alias, err := e.AccountAlias(ctx)
		ttl := aliasCacheTTL
		if err != nil {
			e.Logger.Warn("account alias lookup failed", "err", err)
			ttl = aliasErrorCacheTTL
		}
		if id != "" {
			e.aliasAccount = id
		}
		if err == nil {
			e.aliasValue = alias
		}
		e.aliasExpires = time.Now().Add(ttl)
	}
	if accountID != e.aliasAccount {
		return ""
	}
	return e.aliasValue
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
	c := tagging.NewClient(
		e.Logger,
		nil,
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

// EnrichLine parses one JSON object, adds resource_tags and resource_arn, and
// re-marshals JSON. Always preserves the metric: on association failure
// resource_tags is {} and resource_arn is "".
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

	tags := map[string]string{}
	var arn string
	if res := e.resourceForMetric(ctx, accountID, region, ns, metricName, dimensions); res != nil {
		arn = res.ARN
		for _, t := range res.Tags {
			tags[t.Key] = t.Value
		}
	}
	obj["resource_tags"] = tags
	obj["resource_arn"] = arn
	if alias := e.accountAlias(ctx, accountID); alias != "" {
		obj["account_alias"] = alias
	}

	out, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// resourceForMetric resolves the tagged AWS resource backing a single CloudWatch
// metric. Returns nil when no matching resource is found.
func (e *Enricher) resourceForMetric(ctx context.Context, accountID, region, namespace, metricName string, dimensions map[string]string) *model.TaggedResource {
	entry, err := e.getResources(ctx, accountID, region, namespace)
	if err != nil {
		e.Logger.Warn("getResources failed", "err", err, "namespace", namespace, "region", region)
		return nil
	}
	if entry == nil {
		return nil
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
	return res
}
