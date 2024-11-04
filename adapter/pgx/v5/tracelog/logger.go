package tracelog

import (
	"context"
	"log/slog"
	"slices"
	"sort"

	"github.com/jackc/pgx/v5/tracelog"

	"github.com/networkteam/slogutils"
)

// Logger is an adapter for pgx tracelog to slog
type Logger struct {
	logger       *slog.Logger
	ignoreErrors func(err error) bool
	levelsMap    map[tracelog.LogLevel]slog.Level
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
	lvl, levelOK := l.toLevel(level)
	if !l.logger.Enabled(ctx, lvl) {
		return
	}

	if data["err"] != nil && l.ignoreErrors != nil && l.ignoreErrors(data["err"].(error)) {
		return
	}

	attrs := l.buildAttrs(data)

	if !levelOK {
		attrs = append(attrs, slog.Any("INVALID_PGX_LOG_LEVEL", level))
	}

	l.logger.LogAttrs(ctx, lvl, msg, attrs...)
}

func (l *Logger) buildAttrs(data map[string]any) []slog.Attr {
	sortedKeys := []string{"err", "sql", "args"}

	var additionalKeys []string
	for k := range data {
		if !slices.Contains(sortedKeys, k) {
			additionalKeys = append(additionalKeys, k)
		}
	}
	sort.Strings(additionalKeys)

	allKeys := append(sortedKeys, additionalKeys...)

	var attrs []slog.Attr
	for _, k := range allKeys {
		if v, ok := data[k]; ok {
			attrs = append(attrs, slog.Any(k, v))
		}
	}

	return attrs
}

func (l *Logger) toLevel(level tracelog.LogLevel) (slog.Level, bool) {
	if l.levelsMap != nil {
		if mappedLevel, ok := l.levelsMap[level]; ok {
			return mappedLevel, true
		}
	}
	switch level {
	case tracelog.LogLevelTrace:
		return slogutils.LevelTrace, true
	case tracelog.LogLevelDebug:
		return slog.LevelDebug, true
	case tracelog.LogLevelInfo:
		return slog.LevelInfo, true
	case tracelog.LogLevelWarn:
		return slog.LevelWarn, true
	case tracelog.LogLevelError:
		return slog.LevelError, true
	default:
		return slog.LevelError, false
	}
}

// LoggerOpt sets options for the logger
type LoggerOpt func(*Logger)

// WithIgnoreErrors sets an option to ignore certain errors based on a matcher function
func WithIgnoreErrors(matcher func(err error) bool) LoggerOpt {
	return func(l *Logger) {
		l.ignoreErrors = matcher
	}
}

// WithRemapLevel sets a mapping entry between pgx log levels and slog levels
func WithRemapLevel(in tracelog.LogLevel, out slog.Level) LoggerOpt {
	return func(l *Logger) {
		if l.levelsMap == nil {
			l.levelsMap = make(map[tracelog.LogLevel]slog.Level)
		}
		l.levelsMap[in] = out
	}
}
