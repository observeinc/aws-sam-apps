package logging

import (
	"log/slog"
	"os"

	"github.com/go-logr/logr"
)

type Config struct {
	Verbosity int
}

func New(config *Config) logr.Logger {
	logOptions := slog.HandlerOptions{
		AddSource: true,
		Level:     slog.Level(-config.Verbosity),
	}
	return logr.FromSlogHandler(slog.NewJSONHandler(os.Stderr, &logOptions))
}
