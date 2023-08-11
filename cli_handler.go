package slogutils

import (
	"bytes"
	"context"
	"encoding"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strconv"
	"sync"
	"unicode"

	"github.com/fatih/color"
	"github.com/mattn/go-colorable"
)

var cliDefaultLevelColors = map[slog.Level]*color.Color{
	LevelTrace:      color.New(color.Faint),
	slog.LevelDebug: color.New(color.FgWhite, color.Faint),
	slog.LevelInfo:  color.New(color.FgBlue),
	slog.LevelWarn:  color.New(color.FgYellow),
	slog.LevelError: color.New(color.FgRed),
}

const cliDefaultPrefixPadding = 2

var cliDefaultLevelPrefixes = map[slog.Level]string{
	LevelTrace:      "-",
	slog.LevelDebug: "◦",
	slog.LevelInfo:  "•",
	slog.LevelWarn:  "▲",
	slog.LevelError: "✕",
}

const cliDefaultMessagePadding = 25

// CLIHandlerOptions are options for a CLIHandler.
// A zero CLIHandlerOptions consists entirely of default values.
type CLIHandlerOptions struct {
	// Level reports the minimum record level that will be logged.
	// The handler discards records with lower levels.
	// If Level is nil, the handler assumes LevelInfo.
	// The handler calls Level.Level for each record processed;
	// to adjust the minimum level dynamically, use a LevelVar.
	Level slog.Leveler

	// Prefix options for setting a custom padding and level prefixes.
	Prefix *PrefixOptions

	// LevelColors can set a custom map of level colors.
	// It must be complete, i.e. contain all levels.
	LevelColors map[slog.Level]*color.Color

	// MessagePadding is the number of spaces to pad the message with.
	// A default of 25 is used if this is 0.
	// Setting it to a negative value disables padding.
	MessagePadding int

	// ReplaceAttr is called to rewrite each non-group attribute before it is logged.
	// See https://pkg.go.dev/log/slog#HandlerOptions for details.
	ReplaceAttr func(groups []string, attr slog.Attr) slog.Attr
}

type PrefixOptions struct {
	// Padding is the number of spaces to pad the prefix with.
	Padding int

	// Prefixes can set a custom map of prefixes for each level.
	// It must be complete, i.e. contain all levels.
	Prefixes map[slog.Level]string
}

type CLIHandler struct {
	w    io.Writer
	goas []groupOrAttrs

	level          slog.Leveler
	prefixPadding  int
	levelPrefixes  map[slog.Level]string
	levelColors    map[slog.Level]*color.Color
	replaceAttr    func(groups []string, attr slog.Attr) slog.Attr
	messagePadding int

	mu *sync.Mutex
}

var _ slog.Handler = (*CLIHandler)(nil)

func NewCLIHandler(w io.Writer, opts *CLIHandlerOptions) *CLIHandler {
	if opts == nil {
		opts = &CLIHandlerOptions{}
	}

	if opts.Level == nil {
		opts.Level = slog.LevelInfo
	}

	if opts.Prefix == nil {
		opts.Prefix = &PrefixOptions{
			Padding:  cliDefaultPrefixPadding,
			Prefixes: cliDefaultLevelPrefixes,
		}
	}

	if opts.LevelColors == nil {
		opts.LevelColors = cliDefaultLevelColors
	}

	if opts.MessagePadding == 0 {
		opts.MessagePadding = cliDefaultMessagePadding
	} else if opts.MessagePadding < 0 {
		opts.MessagePadding = 0
	}

	if f, ok := w.(*os.File); ok {
		w = colorable.NewColorable(f)
	}

	return &CLIHandler{
		w: w,

		level:          opts.Level,
		prefixPadding:  opts.Prefix.Padding,
		levelPrefixes:  opts.Prefix.Prefixes,
		levelColors:    opts.LevelColors,
		messagePadding: opts.MessagePadding,
		replaceAttr:    opts.ReplaceAttr,

		mu: &sync.Mutex{},
	}
}

func (h *CLIHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return level >= h.level.Level()
}

func (h *CLIHandler) Handle(ctx context.Context, r slog.Record) error {
	levelColor := cliDefaultLevelColors[r.Level]
	levelPrefix := h.levelPrefixes[r.Level]

	// Note: this handler should not be performance critical, so we don't use a buffer pool or pre-formatting for now.
	buf := new(bytes.Buffer)

	h.mu.Lock()
	defer h.mu.Unlock()

	msg := r.Message
	if h.replaceAttr != nil {
		if a := h.replaceAttr(nil, slog.String(slog.MessageKey, msg)); a.Key != "" {
			msg = a.Value.String()
		} else {
			msg = ""
		}
	}

	_, _ = levelColor.Fprintf(buf, "%*s", h.prefixPadding+1, levelPrefix)
	_, _ = fmt.Fprintf(buf, " %-"+strconv.Itoa(h.messagePadding)+"s", msg)

	// Handle state from WithGroup and WithAttrs.
	goas := h.goas
	if r.NumAttrs() == 0 {
		// If the record has no Attrs, remove groups at the end of the list; they are empty.
		for len(goas) > 0 && goas[len(goas)-1].group != "" {
			goas = goas[:len(goas)-1]
		}
	}

	attrPrefix := ""
	groups := make([]string, 0, len(goas))
	for _, goa := range goas {
		if goa.group != "" {
			attrPrefix += goa.group + "."
			groups = append(groups, goa.group)
		} else {
			for _, a := range goa.attrs {
				if h.replaceAttr != nil {
					a = h.replaceAttr(groups, a)
				}
				h.appendAttr(buf, levelColor, a, attrPrefix)
			}
		}
	}
	r.Attrs(func(a slog.Attr) bool {
		if h.replaceAttr != nil {
			a = h.replaceAttr(groups, a)
		}
		h.appendAttr(buf, levelColor, a, attrPrefix)
		return true
	})

	buf.WriteRune('\n')

	_, _ = buf.WriteTo(h.w)

	return nil
}

func (h *CLIHandler) appendAttr(buf *bytes.Buffer, levelColor *color.Color, attr slog.Attr, groupsPrefix string) {
	if attr.Equal(slog.Attr{}) {
		return
	}

	attr.Value = attr.Value.Resolve()

	switch attr.Value.Kind() {
	case slog.KindGroup:
		groupsPrefix += attr.Key + "."
		for _, groupAttr := range attr.Value.Group() {
			h.appendAttr(buf, levelColor, groupAttr, groupsPrefix)
		}
	default:
		buf.WriteRune(' ')
		levelColor.SetWriter(buf)
		appendString(buf, groupsPrefix+attr.Key, true)
		levelColor.UnsetWriter(buf)
		buf.WriteRune('=')
		appendValue(buf, attr.Value, true)
	}
}

// groupOrAttrs holds either a group name or a list of slog.Attrs.
type groupOrAttrs struct {
	group string      // group name if non-empty
	attrs []slog.Attr // attrs if non-empty
}

func (h *CLIHandler) withGroupOrAttrs(goa groupOrAttrs) *CLIHandler {
	h2 := *h // Copy handler
	h2.goas = make([]groupOrAttrs, len(h.goas)+1)
	copy(h2.goas, h.goas)
	h2.goas[len(h2.goas)-1] = goa
	return &h2
}

func (h *CLIHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}
	return h.withGroupOrAttrs(groupOrAttrs{attrs: attrs})
}

func (h *CLIHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	return h.withGroupOrAttrs(groupOrAttrs{group: name})
}

// Code inspired by github.com/lmittmann/tint

func needsQuoting(s string) bool {
	if len(s) == 0 {
		return true
	}
	for _, r := range s {
		if unicode.IsSpace(r) || r == '"' || r == '=' || !unicode.IsPrint(r) {
			return true
		}
	}
	return false
}

func appendValue(buf *bytes.Buffer, v slog.Value, quote bool) {
	switch v.Kind() {
	case slog.KindString:
		appendString(buf, v.String(), quote)
	case slog.KindInt64:
		buf.WriteString(strconv.FormatInt(v.Int64(), 10))
	case slog.KindUint64:
		buf.WriteString(strconv.FormatUint(v.Uint64(), 10))
	case slog.KindFloat64:
		buf.WriteString(strconv.FormatFloat(v.Float64(), 'g', -1, 64))
	case slog.KindBool:
		buf.WriteString(strconv.FormatBool(v.Bool()))
	case slog.KindDuration:
		appendString(buf, v.Duration().String(), quote)
	case slog.KindTime:
		appendString(buf, v.Time().String(), quote)
	case slog.KindAny:
		if tm, ok := v.Any().(encoding.TextMarshaler); ok {
			data, err := tm.MarshalText()
			if err != nil {
				break
			}
			appendString(buf, string(data), quote)
			break
		}
		appendString(buf, fmt.Sprint(v.Any()), quote)
	}
}

func appendString(buf *bytes.Buffer, s string, quote bool) {
	if quote && needsQuoting(s) {
		buf.WriteString(strconv.Quote(s))
	} else {
		buf.WriteString(s)
	}
}
