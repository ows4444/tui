// Package clockview is a wall-clock / stopwatch / countdown-timer widget. All
// three share the same motion-driven animation mechanism (spinner.Model's
// self-rescheduling tick pattern), and stopwatch/timer share the same
// elapsed-time mechanism — only View and the tick's effect on elapsed differ
// per Mode, so this is one Model type with a Mode field rather than three
// separate types.
package clockview

import (
	"fmt"
	"sync/atomic"
	"time"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/motion"
)

// Mode selects what a Model tracks and renders.
type Mode int

const (
	// ModeClock renders the current wall-clock time.
	ModeClock Mode = iota
	// ModeStopwatch counts elapsed time up from zero.
	ModeStopwatch
	// ModeTimer counts remaining time down from Duration to zero.
	ModeTimer
)

// Model animates a clock, stopwatch or timer depending on Mode. It uses the
// same self-rescheduling motion tick pattern spinner.Model does: each tick's
// Cmd only reschedules itself while still running, so Stop needs no
// explicit cancellation.
type Model struct {
	Mode Mode

	// Duration is the countdown total in ModeTimer; ignored otherwise.
	Duration time.Duration
	// Interval is the tick rate.
	Interval time.Duration
	// Layout is the time.Format layout used in ModeClock.
	Layout string
	// Now returns the current time; injectable for testability. Defaults
	// to time.Now in New.
	Now func() time.Time

	elapsed time.Duration
	running bool
	// gen identifies the live tick chain. Stop, and a Start on a running
	// Model, advance it, so a tick already in flight from the old chain is
	// recognised as stale and dropped instead of running beside the new one.
	gen int
	// id tells this Model's ticks from another's. A program that holds two
	// Models passes every message to both, and without it each would take
	// the other's tick for its own, count it, and schedule one more. Start
	// sets it; a copy of a Model is the same widget and keeps it.
	id uint64
}

// lastID is the id most recently given to a Model by Start.
var lastID atomic.Uint64

// New returns an idle Model in the given Mode with sane defaults: a 1s
// Interval, a "15:04:05" clock Layout and time.Now for Now. Set Duration
// (ModeTimer) or Layout (ModeClock) on the returned Model as needed.
func New(mode Mode) Model {
	return Model{
		Mode:     mode,
		Interval: time.Second,
		Layout:   "15:04:05",
		Now:      time.Now,
	}
}

type tickMsg struct {
	id  uint64
	gen int
}

func tickCmd(d time.Duration, id uint64, gen int) tui.Cmd {
	return tui.FromCtx(motion.After(d, func(time.Time) tui.Msg { return tickMsg{id: id, gen: gen} }))
}

// Start begins the animation; return the Cmd it produces from your own
// Init or Update so it actually runs.
// Call it on the model your program keeps: in an Init with a value receiver
// it would run on a copy, and the kept model would ignore the ticks. Start
// where the model is built and have Init return that Cmd.
func (m *Model) Start() tui.Cmd {
	if m.running {
		m.gen++ // a restart: the chain already ticking is now stale
	}
	if m.id == 0 {
		m.id = lastID.Add(1)
	}
	m.running = true
	return tickCmd(m.Interval, m.id, m.gen)
}

// Stop ends the animation without resetting elapsed. A tick already on its
// way is dropped even if Start is called again before it arrives.
func (m *Model) Stop() {
	m.running = false
	m.gen++
}

// Running reports whether the animation is currently running.
func (m Model) Running() bool { return m.running }

// Update advances the model on each of its own ticks; a no-op for any other
// Msg, another Model's tick included, or while stopped. In ModeTimer, once elapsed reaches Duration it clamps,
// sets Running() false and stops rescheduling — further ticks are no-ops.
func (m Model) Update(msg tui.Msg) (Model, tui.Cmd) {
	tm, ok := msg.(tickMsg)
	if !ok || !m.running || tm.id != m.id || tm.gen != m.gen {
		return m, nil
	}

	switch m.Mode {
	case ModeStopwatch:
		m.elapsed += m.Interval
	case ModeTimer:
		m.elapsed += m.Interval
		if m.elapsed >= m.Duration {
			m.elapsed = m.Duration
			m.running = false
			return m, nil
		}
	case ModeClock:
		// No state to update; a tick just triggers a re-render.
	}
	return m, tickCmd(m.Interval, m.id, m.gen)
}

// View renders the model per its Mode: ModeStopwatch renders elapsed time
// counting up, ModeTimer renders remaining time (Duration - elapsed)
// counting down, both formatted as HH:MM:SS, and ModeClock renders
// m.Now().Format(m.Layout).
func (m Model) View() string {
	switch m.Mode {
	case ModeStopwatch:
		return formatDuration(m.elapsed)
	case ModeTimer:
		remaining := m.Duration - m.elapsed
		if remaining < 0 {
			remaining = 0
		}
		return formatDuration(remaining)
	case ModeClock:
		now := m.Now
		if now == nil {
			now = time.Now
		}
		return now().Format(m.Layout)
	default:
		return ""
	}
}

// formatDuration renders d as HH:MM:SS.
func formatDuration(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	total := int(d / time.Second)
	h := total / 3600
	m := (total % 3600) / 60
	s := total % 60
	return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
}

// Compile-time proof that Model satisfies tui.Component[Model] — see
// tui.Component's doc comment for what this contract means and why.
var _ tui.Component[Model] = Model{}
