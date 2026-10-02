// Package skeleton is a loading placeholder block — InkUI's "Skeleton".
package skeleton

import (
	"strings"
	"time"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/motion"
	"github.com/ows4444/tui/theme"
)

// fillRune is the default placeholder character each row is built from; View
// draws the theme's BarEmpty glyph, which is this one unless the theme is ASCII.
var fillRune = theme.UnicodeGlyphSet().BarEmpty

// shimmerWidth is how many columns of a row are highlighted as the
// "shimmer" sweeps across it.
const shimmerWidth = 3

// Model renders Lines rows of Width placeholder cells, with a shimmer
// highlight that sweeps across each row while running. It uses the same
// self-rescheduling motion tick pattern spinner.Model does: each tick's Cmd
// only reschedules itself while still running, so Stop needs no explicit
// cancellation — a tick that arrives after Stop just doesn't reschedule
// another one.
type Model struct {
	Width int
	Lines int
	Theme theme.Theme

	// tokens is the per-instance colour override set by WithTokens.
	tokens   theme.Tokens
	Interval time.Duration
	// Motion is the reduced-motion preference. The zero value is
	// motion.Normal, the behaviour before this field existed. With
	// motion.Reduced, Start schedules no tick, so the shimmer stays put.
	Motion motion.Preference
	// Clock, if set, drives the animation: the widget renders from the
	// shared Clock's frame and schedules no tick of its own (Start returns
	// nil; start the Clock instead). Nil keeps the per-widget tick.
	Clock *motion.Clock

	frame   int
	running bool
}

// New returns an idle Model at a 100ms interval using theme.DarkTheme().
func New() Model {
	return Model{Interval: 100 * time.Millisecond, Theme: theme.DarkTheme()}
}

type tickMsg struct{}

func tickCmd(d time.Duration) tui.Cmd {
	return tui.FromCtx(motion.After(d, func(time.Time) tui.Msg { return tickMsg{} }))
}

// Start begins the shimmer animation; return the Cmd it produces from your
// own Init or Update so it actually runs.
func (m *Model) Start() tui.Cmd {
	m.running = true
	if m.Motion.Reduced() || m.Clock != nil {
		return nil
	}
	return tickCmd(m.Interval)
}

// Stop ends the animation without resetting its frame.
func (m *Model) Stop() { m.running = false }

// Running reports whether the animation is currently running.
func (m Model) Running() bool { return m.running }

// Update advances the shimmer by one frame; a no-op for any Msg but its
// own tick, or while stopped.
func (m Model) Update(msg tui.Msg) (Model, tui.Cmd) {
	if _, ok := msg.(tickMsg); !ok || !m.running || m.Motion.Reduced() {
		return m, nil
	}
	m.frame++
	return m, tickCmd(m.Interval)
}

// View renders Lines rows, each exactly Width cells wide (measured with
// ansi.Width), filled with fillRune and styled with the Theme's Muted
// color, except for a shimmerWidth-wide highlighted span whose column
// position is a function of m.frame — so consecutive frames sweep the
// highlight across the row. Width <= 0 or Lines <= 0 renders "" without
// panicking.
func (m Model) View() string {
	if m.Width <= 0 || m.Lines <= 0 {
		return ""
	}

	muted := ansi.NewStyle().Foreground(m.themed().Muted)
	fill := m.themed().GlyphSet().BarEmpty
	highlight := ansi.NewStyle().Foreground(m.themed().Text).Bold()

	// The shimmer sweeps across the full row width (including a run-off
	// on either side so it doesn't just snap at the edges) and repeats
	// once it passes the end.
	span := m.Width + shimmerWidth
	frame := m.frame
	if m.Clock != nil {
		frame = m.Clock.Frame()
	}
	pos := frame % span
	start := pos - shimmerWidth

	row := renderRow(m.Width, start, muted, highlight, fill)
	rows := make([]string, m.Lines)
	for i := range rows {
		rows[i] = row
	}
	return strings.Join(rows, "\n")
}

func renderRow(width, highlightStart int, muted, highlight ansi.Style, fill string) string {
	var b strings.Builder
	for i := 0; i < width; i++ {
		if i >= highlightStart && i < highlightStart+shimmerWidth {
			b.WriteString(highlight.Render(fill))
			continue
		}
		b.WriteString(muted.Render(fill))
	}
	return b.String()
}

// Compile-time proof that Model satisfies tui.Component[Model] — see
// tui.Component's doc comment for what this contract means and why.
var _ tui.Component[Model] = Model{}
