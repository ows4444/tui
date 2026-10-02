// Package loadingbar is an indeterminate progress animation — InkUI's
// "LoadingBar": a segment sliding back and forth across a fixed-width
// track, distinct from widgets.ProgressBar's determinate fill.
package loadingbar

import (
	"strings"
	"time"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/motion"
	"github.com/ows4444/tui/theme"
)

// Model animates a segment bouncing across a Width-wide track. Like
// spinner.Model, it uses a self-rescheduling motion tick: Stop needs no
// explicit cancellation, since a tick that arrives after Stop just
// doesn't reschedule another one.
type Model struct {
	Width    int
	Interval time.Duration
	Theme    theme.Theme

	// tokens is the per-instance colour override set by WithTokens.
	tokens theme.Tokens
	// Motion is the reduced-motion preference. The zero value is
	// motion.Normal, the behaviour before this field existed. With
	// motion.Reduced, Start schedules no tick, so the segment stays at its start.
	Motion motion.Preference
	// Clock, if set, drives the animation: the widget renders from the
	// shared Clock's frame and schedules no tick of its own (Start returns
	// nil; start the Clock instead). Nil keeps the per-widget tick.
	Clock *motion.Clock

	pos     int
	dir     int // +1 moving right, -1 moving left
	running bool
}

// New returns an idle Model with an 80ms interval and a width-wide track.
func New(width int) Model {
	return Model{Width: width, Interval: 80 * time.Millisecond, Theme: theme.DarkTheme(), dir: 1}
}

type tickMsg struct{}

func tickCmd(d time.Duration) tui.Cmd {
	return tui.FromCtx(motion.After(d, func(time.Time) tui.Msg { return tickMsg{} }))
}

// Start begins the animation; return the Cmd it produces from your own
// Init or Update.
func (m *Model) Start() tui.Cmd {
	m.running = true
	if m.Motion.Reduced() || m.Clock != nil {
		return nil
	}
	return tickCmd(m.Interval)
}

// Stop ends the animation without resetting its position.
func (m *Model) Stop() { m.running = false }

// Running reports whether the animation is currently running.
func (m Model) Running() bool { return m.running }

// segmentLength is the moving segment's width: a third of the track,
// clamped to at least 1 (and to the track's own width, for very narrow
// bars).
func (m Model) segmentLength() int {
	if m.Width <= 0 {
		return 0
	}
	l := m.Width / 3
	if l < 1 {
		l = 1
	}
	if l > m.Width {
		l = m.Width
	}
	return l
}

func (m Model) maxPos() int {
	maxPos := m.Width - m.segmentLength()
	if maxPos < 0 {
		return 0
	}
	return maxPos
}

// Update advances the segment one step and bounces it off either end of
// the track; a no-op for any Msg but its own tick, or while stopped.
func (m Model) Update(msg tui.Msg) (Model, tui.Cmd) {
	if _, ok := msg.(tickMsg); !ok || !m.running || m.Motion.Reduced() {
		return m, nil
	}

	maxPos := m.maxPos()
	next := m.pos + m.dir
	switch {
	case next >= maxPos:
		next = maxPos
		m.dir = -1
	case next <= 0:
		next = 0
		m.dir = 1
	}
	m.pos = next
	return m, tickCmd(m.Interval)
}

// View renders the track with the moving segment at its current position.
func (m Model) View() string {
	if m.Width <= 0 {
		return ""
	}
	segLen := m.segmentLength()
	pos := clamp(m.pos, 0, m.maxPos())
	if m.Clock != nil {
		pos = bouncePos(m.Clock.Frame(), m.maxPos())
	}

	track := ansi.NewStyle().Foreground(m.themed().Muted)
	segment := ansi.NewStyle().Foreground(m.themed().Primary)

	g := m.themed().GlyphSet()
	var b strings.Builder
	b.WriteString(track.Render(strings.Repeat(g.BarEmpty, pos)))
	b.WriteString(segment.Render(strings.Repeat(g.BarFull, segLen)))
	b.WriteString(track.Render(strings.Repeat(g.BarEmpty, m.Width-pos-segLen)))
	return b.String()
}

// bouncePos is the segment position after n steps of the bounce, matching
// Update's walk from 0 up to maxPos and back.
func bouncePos(n, maxPos int) int {
	if maxPos <= 0 {
		return 0
	}
	n %= 2 * maxPos
	if n > maxPos {
		n = 2*maxPos - n
	}
	return n
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// Compile-time proof that Model satisfies tui.Component[Model] — see
// tui.Component's doc comment for what this contract means and why.
var _ tui.Component[Model] = Model{}

// Linearize renders the indeterminate bar as plain text for accessible
// output (see tui.Linearizer): "Loading, in progress" while running,
// "Loading, stopped" otherwise. The sliding segment carries no information a
// screen reader could use. A zero-width bar reads as "".
func (m Model) Linearize() string {
	if m.Width <= 0 {
		return ""
	}
	if m.running {
		return "Loading, in progress"
	}
	return "Loading, stopped"
}
