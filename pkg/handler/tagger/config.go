package tagger

import (
	"time"

	"github.com/observeinc/aws-sam-apps/pkg/logging"
)

type Config struct {
	OutputFormat string        `env:"OUTPUT_FORMAT,default=json"`
	CacheTTL     time.Duration `env:"CACHE_TTL,default=5m"`
	CachePath    string        `env:"CACHE_PATH,default=/tmp"`

	Logging *logging.Config
}
