package logging

import (
	"log/slog"
	"os"

	"github.com/go-logr/logr"
)

type Config struct {
	Verbosity   int    `env:"VERBOSITY,default=1"`
	AddSource   bool   `env:"LOG_ADD_SOURCE,default=true"`
	HandlerType string `env:"LOG_HANDLER_TYPE,default=json"`
}

func New(config *Config) logr.Logger {
	logOptions := slog.HandlerOptions{
		AddSource: config.AddSource,
		Level:     slog.Level(-config.Verbosity),
	}

	var handler slog.Handler

	switch config.HandlerType {
	case "text":
		handler = slog.NewTextHandler(os.Stderr, &logOptions)
	default:
		handler = slog.NewJSONHandler(os.Stderr, &logOptions)
	}
	return logr.FromSlogHandler(handler)
}
