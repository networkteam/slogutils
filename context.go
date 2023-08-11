package slogutils

import (
	"context"
	"log/slog"
)

type contextKey int

const (
	loggerKey contextKey = iota
)

// FromContext returns a logger instance from the context or the default logger.
func FromContext(ctx context.Context) *slog.Logger {
	v := ctx.Value(loggerKey)
	if l, ok := v.(*slog.Logger); ok {
		return l
	}
	return slog.Default()
}

// WithLogger sets the logger instance as a value in the context and returns the new context.
// The logger instance can be retrieved from the context using FromContext.
func WithLogger(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerKey, logger)
}
