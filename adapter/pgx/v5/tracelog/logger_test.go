package tracelog_test

import (
	"context"
	"fmt"
	"log/slog"
	"testing"

	"github.com/jackc/pgx/v5/tracelog"
	"github.com/vgarvardt/slogex/observer"

	"github.com/networkteam/slogutils"
	logutilstracelog "github.com/networkteam/slogutils/adapter/pgx/v5/tracelog"
)

func TestLogger_Log(t *testing.T) {
	var testErr = fmt.Errorf("test error")

	type args struct {
		level tracelog.LogLevel
		msg   string
		data  map[string]interface{}
	}
	tests := []struct {
		name        string
		applyLogger func(logger *slog.Logger) *slog.Logger
		args        args
		opts        []logutilstracelog.LoggerOpt
		expected    *observer.LoggedRecord
	}{
		{
			name: "pgx trace is logged as trace",
			args: args{
				level: tracelog.LogLevelTrace,
				msg:   "Hey, it's a test",
				data: map[string]interface{}{
					"foo": "bar",
				},
			},
			expected: &observer.LoggedRecord{
				Record: slog.Record{
					Level:   slogutils.LevelTrace,
					Message: "Hey, it's a test",
				},
				Attrs: []slog.Attr{slog.String("foo", "bar")},
			},
		},
		{
			name: "pgx debug is logged as debug",
			args: args{
				level: tracelog.LogLevelDebug,
				msg:   "Hey, it's a test",
				data: map[string]interface{}{
					"foo": "bar",
				},
			},
			expected: &observer.LoggedRecord{
				Record: slog.Record{
					Level:   slog.LevelDebug,
					Message: "Hey, it's a test",
				},
				Attrs: []slog.Attr{slog.String("foo", "bar")},
			},
		},
		{
			name: "pgx info is logged as info",
			args: args{
				level: tracelog.LogLevelInfo,
				msg:   "Hey, it's a test",
				data: map[string]interface{}{
					"foo": "bar",
				},
			},
			expected: &observer.LoggedRecord{
				Record: slog.Record{
					Level:   slog.LevelInfo,
					Message: "Hey, it's a test",
				},
				Attrs: []slog.Attr{slog.String("foo", "bar")},
			},
		},
		{
			name: "pgx warn is logged as warn",
			args: args{
				level: tracelog.LogLevelWarn,
				msg:   "Hey, it's a test",
				data: map[string]interface{}{
					"foo": "bar",
				},
			},
			expected: &observer.LoggedRecord{
				Record: slog.Record{
					Level:   slog.LevelWarn,
					Message: "Hey, it's a test",
				},
				Attrs: []slog.Attr{slog.String("foo", "bar")},
			},
		},
		{
			name: "pgx error is logged as error and included in fields as error",
			args: args{
				level: tracelog.LogLevelError,
				msg:   "Hey, there was an error",
				data: map[string]interface{}{
					"err": testErr,
					"sql": "SELECT * FROM users",
				},
			},
			expected: &observer.LoggedRecord{
				Record: slog.Record{
					Level:   slog.LevelError,
					Message: "Hey, there was an error",
				},
				Attrs: []slog.Attr{
					slog.String("sql", "SELECT * FROM users"),
					slog.Any("err", testErr),
				},
			},
		},
		{
			name: "pgx error is ignored if it matches the ignore error option",
			opts: []logutilstracelog.LoggerOpt{
				logutilstracelog.WithIgnoreErrors(func(err error) bool {
					return err.Error() == "ignored error"
				}),
			},
			args: args{
				level: tracelog.LogLevelError,
				msg:   "Hey, there was an error",
				data: map[string]interface{}{
					"err": fmt.Errorf("ignored error"),
					"sql": "SELECT * FROM users",
				},
			},
			expected: nil,
		},
		{
			name: "logger can be customized",
			applyLogger: func(logger *slog.Logger) *slog.Logger {
				return logger.With("component", "driver.sql")
			},
			args: args{
				level: tracelog.LogLevelInfo,
				msg:   "Hey, it's a test",
				data: map[string]interface{}{
					"foo": "bar",
				},
			},
			expected: &observer.LoggedRecord{
				Record: slog.Record{
					Level:   slog.LevelInfo,
					Message: "Hey, it's a test",
				},
				Attrs: []slog.Attr{slog.String("foo", "bar"), slog.String("component", "driver.sql")},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, observedLogs := observer.New(&observer.HandlerOptions{
				Level: slogutils.LevelTrace,
			})

			logger := slog.New(handler)
			if tt.applyLogger != nil {
				logger = tt.applyLogger(logger)
			}

			p := logutilstracelog.NewLogger(logger, tt.opts...)
			p.Log(context.Background(), tt.args.level, tt.args.msg, tt.args.data)

			logs := observedLogs.All()

			if tt.expected == nil {
				if len(logs) > 0 {
					t.Errorf("expected no log entries, got: %d", len(logs))
				}
				return
			}

			if len(logs) != 1 {
				t.Errorf("Expected 1 entry, got %d", len(logs))
				return
			}

			entry := logs[0]
			if entry.Record.Level != tt.expected.Record.Level {
				t.Errorf("Expected level %s, got %s", tt.expected.Record.Level, entry.Record.Level)
			}
			if entry.Record.Message != tt.expected.Record.Message {
				t.Errorf("Expected message %s, got %s", tt.expected.Record.Message, entry.Record.Message)
			}
			attrs := entry.AttrsMap()
			if len(attrs) != len(tt.expected.Attrs) {
				t.Errorf("Expected %d attrs, got %d: %v", len(tt.expected.Attrs), len(attrs), attrs)
			}
			for k, v := range tt.expected.AttrsMap() {
				if attrs[k] != v {
					t.Errorf("Expected field %s to be %s, got %s", k, v, attrs[k])
				}
			}
		})
	}
}
