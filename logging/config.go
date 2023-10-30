package logging

import (
	"log/slog"
	"os"

	"github.com/go-logr/logr"
	"github.com/go-logr/logr/slogr"
)

type Config struct {
	Verbosity int
}

func New(config *Config) logr.Logger {
	logOptions := slog.HandlerOptions{
		AddSource: true,
		Level:     slog.Level(-config.Verbosity),
	}
	return slogr.NewLogr(slog.NewJSONHandler(os.Stderr, &logOptions))
}
