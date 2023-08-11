package slogutils

import "log/slog"

// Keys for special attributes.
const (
	// ErrorKey is the key for an error attribute.
	ErrorKey = "err"
)

const (
	// LevelTrace is an extra level for tracing that is lower than debug.
	LevelTrace slog.Level = -8
)

func Err(err error) slog.Attr {
	return slog.Attr{Key: ErrorKey, Value: slog.AnyValue(err)}
}
