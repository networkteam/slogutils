package buffering

import (
	"context"
	"log/slog"
	"sync"
)

type Emitter struct {
	mu      sync.Mutex
	records []bufferedRecord
}

type bufferedRecord struct {
	ctx    context.Context
	record slog.Record
	level  slog.Level
	attrs  []slog.Attr
	groups []string
}

type Handler struct {
	*Emitter
	attrs  []slog.Attr
	groups []string
}

func New() *Handler {
	return &Handler{Emitter: &Emitter{}}
}

func (e *Emitter) EmitTo(handler slog.Handler) error {
	if handler == nil {
		return nil
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	for _, br := range e.records {
		current := handler
		// Apply the record's groups and attrs
		for _, g := range br.groups {
			current = current.WithGroup(g)
		}
		if len(br.attrs) > 0 {
			current = current.WithAttrs(br.attrs)
		}

		if current.Enabled(br.ctx, br.level) {
			if err := current.Handle(br.ctx, br.record); err != nil {
				return err
			}
		}
	}

	e.records = nil
	return nil
}

func (h *Handler) Enabled(ctx context.Context, level slog.Level) bool {
	return true
}

func (h *Handler) Handle(ctx context.Context, r slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.records = append(h.records, bufferedRecord{
		ctx:    ctx,
		record: r.Clone(),
		level:  r.Level,
		attrs:  h.attrs,
		groups: h.groups,
	})

	return nil
}

func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &Handler{
		Emitter: h.Emitter,
		attrs:   append(append([]slog.Attr(nil), h.attrs...), attrs...),
		groups:  append([]string(nil), h.groups...),
	}
}

func (h *Handler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	return &Handler{
		Emitter: h.Emitter,
		attrs:   append([]slog.Attr(nil), h.attrs...),
		groups:  append(append([]string(nil), h.groups...), name),
	}
}
