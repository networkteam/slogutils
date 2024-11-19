package buffering_test

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"

	"github.com/networkteam/slogutils/buffering"
)

func TestBufferingHandler(t *testing.T) {
	tests := []struct {
		name   string
		run    func(*buffering.Handler, *slog.Logger)
		want   []string
		minLvl slog.Level
	}{
		{
			name: "basic",
			run: func(h *buffering.Handler, log *slog.Logger) {
				log.Info("msg1", "k1", "v1")
				log.Info("msg2", "k2", "v2")
			},
			want: []string{
				`level=INFO msg=msg1 k1=v1`,
				`level=INFO msg=msg2 k2=v2`,
			},
		},
		{
			name: "with_attrs",
			run: func(h *buffering.Handler, log *slog.Logger) {
				log = log.With("k1", "v1")
				log.Info("msg")
			},
			want: []string{
				`level=INFO msg=msg k1=v1`,
			},
		},
		{
			name: "with_group",
			run: func(h *buffering.Handler, log *slog.Logger) {
				log = log.WithGroup("g1")
				log.Info("msg", "k1", "v1")
			},
			want: []string{
				`level=INFO msg=msg g1.k1=v1`,
			},
		},
		{
			name: "level_filtering",
			run: func(h *buffering.Handler, log *slog.Logger) {
				log.Debug("debug msg")
				log.Info("info msg")
				log.Warn("warn msg")
			},
			minLvl: slog.LevelInfo,
			want: []string{
				`level=INFO msg="info msg"`,
				`level=WARN msg="warn msg"`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			h := buffering.New()
			log := slog.New(h)

			// Run the test
			tt.run(h, log)

			// Emit to text handler
			textHandler := slog.NewTextHandler(&buf, &slog.HandlerOptions{
				Level: tt.minLvl,
			})
			if err := h.EmitTo(textHandler); err != nil {
				t.Fatal(err)
			}

			// Check output
			got := strings.Split(strings.TrimSpace(buf.String()), "\n")
			if len(got) != len(tt.want) {
				t.Fatalf("got %d lines, want %d\n got:\n%s\nwant:\n%s",
					len(got), len(tt.want), buf.String(), strings.Join(tt.want, "\n"))
			}

			for i := range got {
				// Remove time prefix from output
				parts := strings.SplitN(got[i], " level=", 2)
				if len(parts) != 2 {
					t.Errorf("line %d: missing level: %q", i, got[i])
					continue
				}
				got[i] = "level=" + parts[1]

				if got[i] != tt.want[i] {
					t.Errorf("line %d:\ngot:  %s\nwant: %s", i, got[i], tt.want[i])
				}
			}
		})
	}
}
