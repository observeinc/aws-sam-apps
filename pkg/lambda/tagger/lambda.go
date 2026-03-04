package tagger

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/resourcegroupstaggingapi"
	"github.com/go-logr/logr"

	"github.com/observeinc/aws-sam-apps/pkg/handler"
	"github.com/observeinc/aws-sam-apps/pkg/handler/tagger"
	"github.com/observeinc/aws-sam-apps/pkg/logging"
)

type Config struct {
	OutputFormat string `env:"OUTPUT_FORMAT,default=json"`
	CacheTTL     string `env:"CACHE_TTL,default=5m"`
	CachePath    string `env:"CACHE_PATH,default=/tmp"`

	Logging *logging.Config
}

type Lambda struct {
	Logger     logr.Logger
	Entrypoint lambda.Handler
	Shutdown   func()
}

func New(ctx context.Context, cfg *Config) (*Lambda, error) {
	logger := logging.New(cfg.Logging)
	logger.V(4).Info("initialized", "config", cfg)

	l := &Lambda{
		Logger: logger,
		Shutdown: func() {
			logger.V(4).Info("SIGTERM received, running shutdown")
		},
	}

	awsCfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS configuration: %w", err)
	}

	taggingClient := tagger.NewResourceGroupsTaggingClient(
		resourcegroupstaggingapi.NewFromConfig(awsCfg),
	)

	handlerCfg := &tagger.Config{
		OutputFormat: cfg.OutputFormat,
		CachePath:    cfg.CachePath,
		Logging:      cfg.Logging,
	}
	handlerCfg.CacheTTL, err = parseDuration(cfg.CacheTTL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse CACHE_TTL: %w", err)
	}

	cache := tagger.NewTagCache(taggingClient, handlerCfg.CacheTTL, handlerCfg.CachePath, logger)
	h := tagger.New(handlerCfg, cache, logger)

	// Pre-populate cache during init for minimal cold-start latency
	h.Warm(ctx)

	mux := &handler.Mux{
		Logger: logger,
	}

	if err := mux.Register(tagger.WarmHandler(h)); err != nil {
		return nil, fmt.Errorf("failed to register handler: %w", err)
	}

	l.Entrypoint = mux
	return l, nil
}

func parseDuration(s string) (d time.Duration, err error) {
	return time.ParseDuration(s)
}
