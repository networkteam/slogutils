package slogutils_test

import (
	"bytes"
	"context"
	"log/slog"
	"testing"

	"github.com/networkteam/slogutils"
)

func TestFromContext(t *testing.T) {
	ctx := context.Background()

	logger := slogutils.FromContext(ctx)
	if logger == nil {
		t.Fatal("logger from context should not be nil")
	}
	if logger != slog.Default() {
		t.Fatal("logger from context should be the default logger")
	}
}

func TestWithLogger(t *testing.T) {
	ctx := context.Background()

	buf := new(bytes.Buffer)
	logger := slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{
		ReplaceAttr: drop(slog.TimeKey),
	})).
		With("component", "test")
	ctx = slogutils.WithLogger(ctx, logger)

	if logger != slogutils.FromContext(ctx) {
		t.Fatal("logger from context should be the logger set in the context")
	}

	// Make sure we can use the logger
	logger.Info("Just a test")
	if buf.String() != "level=INFO msg=\"Just a test\" component=test\n" {
		t.Fatalf("unexpected log output: %s", buf.String())
	}
}
