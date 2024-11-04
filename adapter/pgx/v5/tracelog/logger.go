package tracelog

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5/tracelog"

	"github.com/networkteam/slogutils"
)

// Logger is an adapter for pgx tracelog to slog
type Logger struct {
	logger       *slog.Logger
	ignoreErrors func(err error) bool
}

// NewLogger builds a new logger instance given a slog.Logger instance
func NewLogger(logger *slog.Logger, opts ...LoggerOpt) *Logger {
	l := &Logger{logger: logger}
	for _, opt := range opts {
		opt(l)
	}
	return l
}

// Log a pgx log message to the underlying log instance, implements tracelog.Logger
func (l *Logger) Log(ctx context.Context, level tracelog.LogLevel, msg string, data map[string]interface{}) {
	unknownLogLevel := false

	var lvl slog.Level
	switch level {
	case tracelog.LogLevelTrace:
		lvl = slogutils.LevelTrace
	case tracelog.LogLevelDebug:
		lvl = slog.LevelDebug
	case tracelog.LogLevelInfo:
		lvl = slog.LevelInfo
	case tracelog.LogLevelWarn:
		lvl = slog.LevelWarn
	case tracelog.LogLevelError:
		lvl = slog.LevelError
	default:
		lvl = slog.LevelError
		unknownLogLevel = true
	}

	if !l.logger.Enabled(ctx, lvl) {
		return
	}

	if data["err"] != nil && l.ignoreErrors != nil && l.ignoreErrors(data["err"].(error)) {
		return
	}

	attrs := make([]slog.Attr, 0, len(data))
	for k, v := range data {
		attrs = append(attrs, slog.Any(k, v))
	}

	if unknownLogLevel {
		attrs = append(attrs, slog.Any("INVALID_PGX_LOG_LEVEL", level))
	}

	l.logger.LogAttrs(ctx, lvl, msg, attrs...)
}

// LoggerOpt sets options for the logger
type LoggerOpt func(*Logger)

func WithIgnoreErrors(matcher func(err error) bool) LoggerOpt {
	return func(l *Logger) {
		l.ignoreErrors = matcher
	}
}
