// Package cellcheck runs a model headless under the cell renderer and reports
// the frames that fell back to the line renderer, for the example corpus tests
// (spec S11: the cell renderer's fallback rate on the examples is zero).
package cellcheck

import (
	"io"
	"strings"
	"sync"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
)

type doneMsg struct{}

// host sizes the terminal, lets the model draw, then quits.
type host struct {
	inner tui.Model
	w, h  int
}

func (h host) Init() tui.Cmd {
	return tui.Sequence(
		func() tui.Msg { return tui.ResizeMsg{Width: h.w, Height: h.h} },
		func() tui.Msg { return doneMsg{} },
	)
}

func (h host) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	if _, ok := msg.(doneMsg); ok {
		return h, tui.Quit()
	}
	next, cmd := h.inner.Update(msg)
	h.inner = next
	return h, cmd
}

func (h host) View() string { return h.inner.View() }

type syncBuf struct {
	mu sync.Mutex
	b  strings.Builder
}

func (s *syncBuf) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Write(p)
}

func (s *syncBuf) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.String()
}

type staticView string

func (v staticView) Init() tui.Cmd                       { return nil }
func (v staticView) Update(tui.Msg) (tui.Model, tui.Cmd) { return v, nil }
func (v staticView) View() string                        { return string(v) }

// FallbacksView is Fallbacks for an already-rendered view string.
func FallbacksView(view string, w, h int) []string {
	return Fallbacks(staticView(view), w, h)
}

// Fallbacks draws m once in a w x h terminal with the cell renderer and
// returns the frame log lines of the frames that fell back, whole-frame
// (kind=fallback, ending in "reason=<slug>") or per row (a fallback_rows=
// field); empty means none did.
func Fallbacks(m tui.Model, w, h int) []string {
	var log syncBuf
	pr, pw := io.Pipe()
	defer func() { _ = pw.Close() }()
	p := tui.NewProgram(host{inner: m, w: w, h: h},
		tui.WithAltScreen(true),
		tui.WithInput(pr),
		tui.WithOutput(io.Discard),
		tui.WithErrOutput(io.Discard),
		tui.WithColorProfile(ansi.TrueColor),
		tui.WithCellRenderer(true),
		tui.WithFrameLog(&log))
	_, _ = p.Run()
	_ = pr.Close()
	var out []string
	if !strings.Contains(log.String(), "kind=") {
		return []string{"no frame was drawn"}
	}
	for _, l := range strings.Split(log.String(), "\n") {
		if strings.Contains(l, "kind=fallback") || strings.Contains(l, "fallback_rows=") {
			out = append(out, l)
		}
	}
	return out
}
