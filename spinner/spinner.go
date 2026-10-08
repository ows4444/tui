// Package spinner is an animated loading indicator — InkUI's "Spinner".
package spinner

import (
	"sync/atomic"
	"time"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/motion"
	"github.com/ows4444/tui/theme"
)

// defaultFrames is a braille-dot spin animation, the common default for a
// terminal spinner. It is never handed out: Frames returns a copy.
var defaultFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// Frames returns a braille-dot spin animation, the common default for a
// terminal spinner, as a fresh slice the caller may change.
func Frames() []string { return append([]string(nil), defaultFrames...) }

// Model animates through Frames at Interval while running. It uses the
// same self-rescheduling motion tick pattern textinput's cursor blink does:
// each tick's Cmd only reschedules itself while still running, so Stop
// needs no explicit cancellation — a tick that arrives after Stop just
// doesn't reschedule another one.
type Model struct {
	Frames   []string
	Interval time.Duration
	Theme    theme.Theme

	// tokens is the per-instance colour override set by WithTokens.
	tokens theme.Tokens
	Label  string
	// Motion is the reduced-motion preference. The zero value is
	// motion.Normal, the behaviour before this field existed. With
	// motion.Reduced, Start schedules no tick, so the spinner stays on its first frame.
	Motion motion.Preference
	// Clock, if set, drives the animation: the widget renders from the
	// shared Clock's frame and schedules no tick of its own (Start returns
	// nil; start the Clock instead). Nil keeps the per-widget tick.
	Clock *motion.Clock

	frame   int
	running bool
	// id tells this Model's ticks from another's. A program that holds two
	// Models passes every message to both, and without it each would take
	// the other's tick for its own, advance, and schedule one more. Start
	// sets it; a copy of a Model is the same widget and keeps it.
	id uint64
}

// lastID is the id most recently given to a Model by Start.
var lastID atomic.Uint64

// New returns an idle Model using Frames() at a 100ms interval.
func New() Model {
	return Model{Frames: Frames(), Interval: 100 * time.Millisecond, Theme: theme.DarkTheme()}
}

type tickMsg struct{ id uint64 }

func tickCmd(d time.Duration, id uint64) tui.Cmd {
	return tui.FromCtx(motion.After(d, func(time.Time) tui.Msg { return tickMsg{id: id} }))
}

// Start begins the animation; return the Cmd it produces from your own
// Init or Update so it actually runs.
// Call it on the model your program keeps: in an Init with a value receiver
// it would run on a copy, and the kept model would ignore the ticks. Start
// where the model is built and have Init return that Cmd.
func (m *Model) Start() tui.Cmd {
	m.running = true
	if m.id == 0 {
		m.id = lastID.Add(1)
	}
	if m.Motion.Reduced() || m.Clock != nil {
		return nil
	}
	return tickCmd(m.Interval, m.id)
}

// Stop ends the animation without resetting its frame.
func (m *Model) Stop() { m.running = false }

// Running reports whether the animation is currently running.
func (m Model) Running() bool { return m.running }

// Update advances to the next frame; a no-op for any Msg but its own
// tick, another Model's tick included, while stopped, or with no Frames.
func (m Model) Update(msg tui.Msg) (Model, tui.Cmd) {
	if t, ok := msg.(tickMsg); !ok || t.id != m.id || !m.running || len(m.Frames) == 0 || m.Motion.Reduced() {
		return m, nil
	}
	m.frame = (m.frame + 1) % len(m.Frames)
	return m, tickCmd(m.Interval, m.id)
}

// isDefaultFrames reports whether f holds exactly the stock frames.
func isDefaultFrames(f []string) bool {
	if len(f) != len(defaultFrames) {
		return false
	}
	for i := range f {
		if f[i] != defaultFrames[i] {
			return false
		}
	}
	return true
}

// View renders the current frame, or Label if Frames is empty.
func (m Model) View() string {
	frames := m.Frames
	if isDefaultFrames(frames) {
		// The stock frames follow the theme's glyph set (ASCII on request);
		// custom frames are drawn as given.
		frames = m.themed().GlyphSet().SpinnerFrames()
	}
	if len(frames) == 0 {
		return m.Label
	}
	// Modulo rather than a direct index: if Frames is swapped for a
	// shorter slice after frame has advanced past its new length, this
	// stays in bounds instead of panicking.
	idx := m.frame
	if m.Clock != nil {
		idx = m.Clock.Frame()
	}
	frame := ansi.NewStyle().Foreground(m.themed().Primary).Render(frames[idx%len(frames)])
	if m.Label == "" {
		return frame
	}
	return frame + " " + m.Label
}

// Compile-time proof that Model satisfies tui.Component[Model] — see
// tui.Component's doc comment for what this contract means and why.
var _ tui.Component[Model] = Model{}
