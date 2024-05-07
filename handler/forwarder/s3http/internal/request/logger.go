package request

import (
	"github.com/go-logr/logr"
)

// leveledLogger provides an adapter between logr.Logger and retryablehttp.LeveledLogger.
type leveledLogger struct {
	logr.Logger
}

func (l *leveledLogger) Error(msg string, keysAndValues ...interface{}) {
	l.V(1).Info(msg, keysAndValues...)
}

func (l *leveledLogger) Warn(msg string, keysAndValues ...interface{}) {
	l.V(2).Info(msg, keysAndValues...)
}

func (l *leveledLogger) Info(msg string, keysAndValues ...interface{}) {
	l.V(3).Info(msg, keysAndValues...)
}

func (l *leveledLogger) Debug(msg string, keysAndValues ...interface{}) {
	l.V(4).Info(msg, keysAndValues...)
}
