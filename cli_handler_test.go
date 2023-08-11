package slogutils_test

import (
	"bytes"
	"errors"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/networkteam/slogutils"
)

func Example() {
	slog.SetDefault(slog.New(slogutils.NewCLIHandler(os.Stderr, &slogutils.CLIHandlerOptions{
		Level: slog.LevelDebug,
	})))

	slog.Info("Starting server", "addr", ":8080", "env", "production")
	slog.Debug("Connected to DB", "db", "myapp", "host", "localhost:5432")
	slog.Warn("Slow request", "method", "GET", "path", "/users", "duration", 497*time.Millisecond)
	slog.Error("DB connection lost", slogutils.Err(errors.New("connection reset")), "db", "myapp")
	// Output:
}

func TestCLIHandler(t *testing.T) {
	tests := []struct {
		Opts *slogutils.CLIHandlerOptions
		F    func(l *slog.Logger)
		Want string
	}{
		{
			F: func(l *slog.Logger) {
				l.Info("test", "key", "val")
			},
			Want: `  • test                      key=val`,
		},
		{
			F: func(l *slog.Logger) {
				l.Error("test", slogutils.Err(errors.New("fail")))
			},
			Want: `  ✕ test                      err=fail`,
		},
		{
			F: func(l *slog.Logger) {
				l.Info("test", slog.Group("group", slog.String("key", "val"), slogutils.Err(errors.New("fail"))))
			},
			Want: `  • test                      group.key=val group.err=fail`,
		},
		{
			F: func(l *slog.Logger) {
				l.WithGroup("group").Info("test", "key", "val")
			},
			Want: `  • test                      group.key=val`,
		},
		{
			F: func(l *slog.Logger) {
				l.With("key", "val").Info("test", "key2", "val2")
			},
			Want: `  • test                      key=val key2=val2`,
		},
		{
			F: func(l *slog.Logger) {
				l.Info("test", "k e y", "v a l")
			},
			Want: `  • test                      "k e y"="v a l"`,
		},
		{
			F: func(l *slog.Logger) {
				l.WithGroup("g r o u p").Info("test", "key", "val")
			},
			Want: `  • test                      "g r o u p.key"=val`,
		},
		{
			F: func(l *slog.Logger) {
				l.Info("test", "slice", []string{"a", "b", "c"}, "map", map[string]int{"a": 1, "b": 2, "c": 3})
			},
			Want: `  • test                      slice="[a b c]" map="map[a:1 b:2 c:3]"`,
		},
		{
			Opts: &slogutils.CLIHandlerOptions{
				ReplaceAttr: drop("bar"),
			},
			F: func(l *slog.Logger) {
				l.Info("test", "foo", "bar", "bar", "baz")
			},
			Want: `  • test                      foo=bar`,
		},
		{
			Opts: &slogutils.CLIHandlerOptions{
				ReplaceAttr: drop(slog.MessageKey),
			},
			F: func(l *slog.Logger) {
				l.Info("test", "key", "val")
			},
			Want: `  •                           key=val`,
		},
		{
			Opts: &slogutils.CLIHandlerOptions{
				ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
					if a.Key == "key" && len(groups) == 1 && groups[0] == "group" {
						return slog.Attr{}
					}
					return a
				},
			},
			F: func(l *slog.Logger) {
				l.WithGroup("group").Info("test", "key", "val", "key2", "val2")
			},
			Want: `  • test                      group.key2=val2`,
		},
		{
			Opts: &slogutils.CLIHandlerOptions{
				ReplaceAttr: replace(slog.IntValue(42), slog.MessageKey),
			},
			F: func(l *slog.Logger) {
				l.Info("test", "key", "val")
			},
			Want: `  • 42                        key=val`,
		},
		{
			Opts: &slogutils.CLIHandlerOptions{
				ReplaceAttr: replace(slog.IntValue(42), "key"),
			},
			F: func(l *slog.Logger) {
				l.With("key", "val").Info("test", "key2", "val2")
			},
			Want: `  • test                      key=42 key2=val2`,
		},
		{
			Opts: &slogutils.CLIHandlerOptions{
				ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
					return slog.Attr{}
				},
			},
			F: func(l *slog.Logger) {
				l.Info("test", "key", "val")
			},
			Want: `  •                          `,
		},
		{
			F: func(l *slog.Logger) {
				l.Info("test", "key", "")
			},
			Want: `  • test                      key=""`,
		},
		{
			F: func(l *slog.Logger) {
				l.Info("test", "", "val")
			},
			Want: `  • test                      ""=val`,
		},
		{
			F: func(l *slog.Logger) {
				l.Info("test", "", "")
			},
			Want: `  • test                      ""=""`,
		},
		{
			F: func(l *slog.Logger) {
				l.Error("test", slog.Any("error", errors.New("fail")))
			},
			Want: `  ✕ test                      error=fail`,
		},
		{
			F: func(l *slog.Logger) {
				l.Error("test", slogutils.Err(nil))
			},
			Want: `  ✕ test                      err=<nil>`,
		},
		{
			Opts: &slogutils.CLIHandlerOptions{
				MessagePadding: -1,
			},
			F: func(l *slog.Logger) {
				l.Info("test", "foo", "bar")
			},
			Want: `  • test foo=bar`,
		},
		{
			Opts: &slogutils.CLIHandlerOptions{
				MessagePadding: -1,
			},
			F: func(l *slog.Logger) {
				l.Info("", "foo", "bar")
			},
			Want: `  •  foo=bar`,
		},
	}

	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			var buf bytes.Buffer
			// test.Opts.NoColor = true
			l := slog.New(slogutils.NewCLIHandler(&buf, test.Opts))
			test.F(l)

			got := strings.TrimRight(buf.String(), "\n")
			if test.Want != got {
				t.Fatalf("(-want +got)\n- %s\n+ %s", test.Want, got)
			}
		})
	}
}

// drop returns a ReplaceAttr that drops the given keys.
func drop(keys ...string) func([]string, slog.Attr) slog.Attr {
	return func(groups []string, a slog.Attr) slog.Attr {
		if len(groups) > 0 {
			return a
		}

		for _, key := range keys {
			if a.Key == key {
				a = slog.Attr{}
			}
		}
		return a
	}
}

func replace(new slog.Value, keys ...string) func([]string, slog.Attr) slog.Attr {
	return func(groups []string, a slog.Attr) slog.Attr {
		if len(groups) > 0 {
			return a
		}

		for _, key := range keys {
			if a.Key == key {
				a.Value = new
			}
		}
		return a
	}
}
