// Package tuislog is a log/slog Handler that writes one styled line per
// record: the time, the level in its theme colour, the message and the
// attributes as key=value. It makes slog safe to use while a Program owns the
// terminal. ToProgram commits each line above the live region, as tui.Println
// does; New hands each line to a function of your choice, such as a
// logview.Model.Append.
//
// Do not log through ToProgram from Update or View: Program.Send must not be
// called there, and a full queue would block the loop. Log from a Cmd or a
// goroutine, or give New a sink that does not block. A line logged after Run
// has returned is dropped.
//
// Message and attribute text is sanitised, so a log record cannot carry escape
// sequences onto the screen.
//
// Stability: experimental. Its API may change in any minor release.
package tuislog

import (
	"context"
	"log/slog"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
)

// Options configures a Handler. The zero value logs Info and above with the
// time as 15:04:05 in the dark theme.
type Options struct {
	// Level is the minimum level logged. Nil means slog.LevelInfo.
	Level slog.Leveler
	// AddSource adds the file and line of the call as file.go:12.
	AddSource bool
	// TimeFormat is the layout of the time column. Empty means "15:04:05";
	// "-" leaves the time out.
	TimeFormat string
	// Theme colours the levels: Error, Warning, Info and Muted for debug. The
	// zero Theme is the dark theme.
	Theme theme.Theme
}

// Handler is a slog.Handler that formats records as styled lines.
type Handler struct {
	sink   func(string)
	opts   Options
	prefix string // the groups opened by WithGroup, as "a.b."
	attrs  string // the attributes added by WithAttrs, already formatted
}

// New returns a Handler that calls sink with each formatted line, without a
// trailing newline. sink may be called from any goroutine.
func New(sink func(line string), o *Options) *Handler {
	h := &Handler{sink: sink}
	if o != nil {
		h.opts = *o
	}
	if h.opts.Theme == (theme.Theme{}) {
		h.opts.Theme = theme.DarkTheme()
	}
	return h
}

// ToProgram returns a Handler that commits each line to p's scrollback above
// the live region, the way tui.Println does. It is safe to use before p.Run
// returns from any goroutine; a line logged before the loop starts waits for
// it, as Program.Send does.
func ToProgram(p *tui.Program, o *Options) *Handler {
	return New(func(line string) { p.Send(tui.Println(line)()) }, o)
}

// Enabled reports whether a record of the given level is logged.
func (h *Handler) Enabled(_ context.Context, l slog.Level) bool {
	min := slog.LevelInfo
	if h.opts.Level != nil {
		min = h.opts.Level.Level()
	}
	return l >= min
}

// Handle formats r and passes it to the sink.
func (h *Handler) Handle(_ context.Context, r slog.Record) error {
	t := h.opts.Theme
	muted := ansi.NewStyle().Foreground(t.Muted)
	var b strings.Builder
	if f := h.opts.TimeFormat; f != "-" && !r.Time.IsZero() {
		if f == "" {
			f = "15:04:05"
		}
		b.WriteString(muted.Render(r.Time.Format(f)))
		b.WriteByte(' ')
	}
	b.WriteString(h.levelStyle(r.Level).Render(levelName(r.Level)))
	b.WriteByte(' ')
	b.WriteString(ansi.Sanitize(r.Message))
	if h.opts.AddSource && r.PC != 0 {
		fr, _ := runtime.CallersFrames([]uintptr{r.PC}).Next()
		b.WriteString(muted.Render(" " + filepath.Base(fr.File) + ":" + strconv.Itoa(fr.Line)))
	}
	var attrs strings.Builder
	attrs.WriteString(h.attrs)
	r.Attrs(func(a slog.Attr) bool {
		writeAttr(&attrs, h.prefix, a)
		return true
	})
	if attrs.Len() > 0 {
		b.WriteString(muted.Render(attrs.String()))
	}
	h.sink(b.String())
	return nil
}

// WithAttrs returns a Handler that adds attrs to every record.
func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	c := *h
	var b strings.Builder
	b.WriteString(h.attrs)
	for _, a := range attrs {
		writeAttr(&b, h.prefix, a)
	}
	c.attrs = b.String()
	return &c
}

// WithGroup returns a Handler that puts later attributes under name.
func (h *Handler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	c := *h
	c.prefix = h.prefix + ansi.Sanitize(name) + "."
	return &c
}

func (h *Handler) levelStyle(l slog.Level) ansi.Style {
	t := h.opts.Theme
	switch {
	case l >= slog.LevelError:
		return ansi.NewStyle().Bold().Foreground(t.Error)
	case l >= slog.LevelWarn:
		return ansi.NewStyle().Bold().Foreground(t.Warning)
	case l >= slog.LevelInfo:
		return ansi.NewStyle().Foreground(t.Info)
	}
	return ansi.NewStyle().Foreground(t.Muted)
}

// levelName is the level as four or five capitals padded to five columns.
func levelName(l slog.Level) string {
	s := l.String()
	if len(s) > 5 {
		s = s[:5]
	}
	return s + strings.Repeat(" ", 5-len(s))
}

// writeAttr appends " key=value" for a, recursing into groups.
func writeAttr(b *strings.Builder, prefix string, a slog.Attr) {
	a.Value = a.Value.Resolve()
	if a.Equal(slog.Attr{}) {
		return
	}
	if a.Value.Kind() == slog.KindGroup {
		p := prefix
		if a.Key != "" {
			p += ansi.Sanitize(a.Key) + "."
		}
		for _, g := range a.Value.Group() {
			writeAttr(b, p, g)
		}
		return
	}
	b.WriteByte(' ')
	b.WriteString(prefix)
	b.WriteString(ansi.Sanitize(a.Key))
	b.WriteByte('=')
	b.WriteString(quote(ansi.Sanitize(valueString(a.Value))))
}

func valueString(v slog.Value) string {
	if v.Kind() == slog.KindTime {
		return v.Time().Format(time.RFC3339)
	}
	return v.String()
}

// quote quotes s when it is empty or holds a space, '=' or '"', so the line
// can be read back as key=value pairs.
func quote(s string) string {
	if s == "" || strings.ContainsAny(s, " =\"\n\t") {
		return strconv.Quote(s)
	}
	return s
}

var _ slog.Handler = (*Handler)(nil)
