// Package slider is a value chosen from a range by moving a thumb along a
// track: Left and Right move it one step, Page Up and Page Down a larger
// one, Home and End to either end. With the mouse on, a press or a drag on
// the track puts the thumb under the pointer.
//
// Stability: experimental. Its API may change in any minor release.
package slider

import (
	"math"
	"strconv"
	"strings"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/hittest"
	"github.com/ows4444/tui/keymap"
	"github.com/ows4444/tui/theme"
)

// Model is one slider. The zero value is not ready to use; build one with
// New.
type Model struct {
	// ID is carried by ChangedMsg, to tell one slider's change from
	// another's.
	ID string
	// Min and Max are the ends of the range. A Max below Min is read as Min.
	Min, Max float64
	// Step is how far Left and Right move the value, and what the value is
	// rounded to. A Step of 0 or less is read as 1.
	Step float64
	// BigStep is how far Page Up and Page Down move the value. A BigStep of
	// 0 or less is read as ten Steps.
	BigStep float64
	// Width is the length of the track in cells. A Width under 2 is read as
	// 2.
	Width int
	// ShowValue draws the value after the track.
	ShowValue bool
	// Format turns the value into the text ShowValue draws and Linearize
	// reads. Nil prints the shortest decimal that is exact.
	Format func(float64) string
	// Disabled stops the value being changed; the slider is drawn dimmed.
	Disabled bool
	Theme    theme.Theme

	// tokens is the per-instance colour override set by WithTokens.
	tokens theme.Tokens
	// KeyMap holds the keys for each action. New fills it with DefaultKeyMap;
	// a Model built as a struct literal with a zero KeyMap behaves as if it
	// held DefaultKeyMap.
	KeyMap KeyMap

	// Mouse, when true, makes Update handle tui.MouseEvent: a left press on
	// the track moves the thumb there, and it follows the pointer until the
	// button comes up. Off (the default) ignores the mouse.
	Mouse bool
	// Bounds is the screen rectangle where the app draws the slider. The
	// track starts one cell in from its left edge.
	Bounds hittest.Rect

	value    float64
	focused  bool
	dragging bool
}

// New returns a slider over min to max with a Step of 1, a track of 20
// cells, and its value at min.
func New(min, max float64) Model {
	return Model{Min: min, Max: max, Step: 1, Width: 20, Theme: theme.DarkTheme(), KeyMap: DefaultKeyMap(), value: min}
}

// KeyMap names the keys of each action of a Model.
type KeyMap struct {
	Dec    keymap.Binding // lower the value by Step
	Inc    keymap.Binding // raise the value by Step
	DecBig keymap.Binding // lower the value by BigStep
	IncBig keymap.Binding // raise the value by BigStep
	ToMin  keymap.Binding // set the value to Min
	ToMax  keymap.Binding // set the value to Max
}

// DefaultKeyMap returns the arrow keys, Page Up and Page Down, Home and End.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Dec:    keymap.NewBinding("lower", "left", "down"),
		Inc:    keymap.NewBinding("raise", "right", "up"),
		DecBig: keymap.NewBinding("lower more", "pgdown"),
		IncBig: keymap.NewBinding("raise more", "pgup"),
		ToMin:  keymap.NewBinding("lowest", "home"),
		ToMax:  keymap.NewBinding("highest", "end"),
	}
}

func (m Model) keys() KeyMap {
	km := m.KeyMap
	if len(km.Dec.Keys)+len(km.Inc.Keys)+len(km.DecBig.Keys)+len(km.IncBig.Keys)+len(km.ToMin.Keys)+len(km.ToMax.Keys) == 0 {
		return DefaultKeyMap()
	}
	return km
}

// Bindings returns the actions the slider honours, with descriptions, for
// help text. A disabled slider has none.
func (m Model) Bindings() []keymap.Binding {
	if m.Disabled {
		return nil
	}
	km := m.keys()
	return []keymap.Binding{km.Dec, km.Inc, km.DecBig, km.IncBig, km.ToMin, km.ToMax}
}

// ChangedMsg is delivered (via the Cmd Update returns) when the value
// changes.
type ChangedMsg struct {
	// ID is the slider's ID.
	ID string
	// Value is the new value.
	Value float64
}

func (m Model) max() float64  { return math.Max(m.Min, m.Max) }
func (m Model) step() float64 { return positive(m.Step, 1) }
func (m Model) big() float64  { return positive(m.BigStep, 10*m.step()) }
func (m Model) width() int    { return max(m.Width, 2) }

func positive(v, fallback float64) float64 {
	if v > 0 {
		return v
	}
	return fallback
}

// snap holds v inside the range and rounds it to a whole number of Steps
// from Min. Max is always reachable, even when the range is not a whole
// number of Steps.
func (m Model) snap(v float64) float64 {
	if math.IsNaN(v) || v <= m.Min {
		return m.Min
	}
	if v >= m.max() {
		return m.max()
	}
	s := m.step()
	v = m.Min + math.Round((v-m.Min)/s)*s
	return math.Min(v, m.max())
}

// Value returns the slider's value.
func (m Model) Value() float64 { return m.snap(m.value) }

// SetValue sets the value, held inside the range and rounded to a Step,
// without a key press: no ChangedMsg is delivered.
func (m *Model) SetValue(v float64) { m.value = m.snap(v) }

// Focus gives the slider keyboard focus. It returns no Cmd; the result is
// there so a slider can stand where any focusable widget does.
func (m *Model) Focus() tui.Cmd {
	m.focused = true
	return nil
}

// Blur takes keyboard focus away, and lets go of a drag in progress.
func (m *Model) Blur() { m.focused, m.dragging = false, false }

// Focused reports whether the slider has keyboard focus.
func (m Model) Focused() bool { return m.focused }

// Dragging reports whether the pointer is holding the thumb.
func (m Model) Dragging() bool { return m.dragging }

// set moves the value to v and returns the Cmd that reports it, or nil when
// the value did not change.
func (m Model) set(v float64) (Model, tui.Cmd) {
	v = m.snap(v)
	if v == m.Value() {
		return m, nil
	}
	m.value = v
	msg := ChangedMsg{ID: m.ID, Value: v}
	return m, func() tui.Msg { return msg }
}

// Update changes the value on the arrow keys, Page Up and Page Down, Home
// and End when the slider has focus, and, with Mouse on, on a press or a
// drag along the track. A disabled slider ignores both.
func (m Model) Update(msg tui.Msg) (Model, tui.Cmd) {
	if ev, ok := msg.(tui.MouseEvent); ok {
		return m.updateMouse(ev)
	}
	if !m.focused || m.Disabled {
		return m, nil
	}
	km := m.keys()
	v := m.Value()
	switch {
	case keymap.Matches(msg, km.Dec):
		return m.set(v - m.step())
	case keymap.Matches(msg, km.Inc):
		return m.set(v + m.step())
	case keymap.Matches(msg, km.DecBig):
		return m.set(v - m.big())
	case keymap.Matches(msg, km.IncBig):
		return m.set(v + m.big())
	case keymap.Matches(msg, km.ToMin):
		return m.set(m.Min)
	case keymap.Matches(msg, km.ToMax):
		return m.set(m.max())
	}
	return m, nil
}

// valueAt is the value of track cell x, where 0 is the first cell.
func (m Model) valueAt(x int) float64 {
	last := m.width() - 1
	x = min(max(x, 0), last)
	return m.Min + (m.max()-m.Min)*float64(x)/float64(last)
}

func (m Model) updateMouse(ev tui.MouseEvent) (Model, tui.Cmd) {
	if !m.Mouse || m.Disabled {
		return m, nil
	}
	x := ev.X - m.Bounds.X - 1 // the track starts after the left bracket cell
	switch ev.Action {
	case tui.MouseActionPress:
		if ev.Button != tui.MouseButtonLeft || !m.Bounds.Contains(ev.X, ev.Y) {
			return m, nil
		}
		m.dragging = true
		return m.set(m.valueAt(x))
	case tui.MouseActionMotion:
		if m.dragging {
			// The thumb follows the pointer along the track wherever the
			// pointer has gone, above, below or past either end.
			return m.set(m.valueAt(x))
		}
	case tui.MouseActionRelease:
		m.dragging = false
	}
	return m, nil
}

// thumb is the track cell the thumb is on.
func (m Model) thumb() int {
	span := m.max() - m.Min
	if span <= 0 {
		return 0
	}
	return int(math.Round((m.Value() - m.Min) / span * float64(m.width()-1)))
}

// text is the value as ShowValue draws it.
func (m Model) text() string {
	if m.Format != nil {
		return ansi.Sanitize(m.Format(m.Value()))
	}
	return strconv.FormatFloat(m.Value(), 'f', -1, 64)
}

// View renders the track on one row: the part below the value filled, the
// thumb, and the rest empty, between two cells that hold angle brackets when the
// slider has focus and blanks when it does not, so focus reads without
// colour and does not change the width. A disabled slider is dimmed and
// held in parentheses. ShowValue adds the value after a space.
func (m Model) View() string {
	t := m.themed()
	st := t.ResolvedStates()
	g := t.GlyphSet()
	at := m.thumb()
	filled := ansi.NewStyle().Foreground(t.Primary)
	rest := ansi.NewStyle().Foreground(t.Muted)
	open, shut := " ", " "
	edge := ansi.NewStyle().Foreground(t.Text)
	switch {
	case m.Disabled:
		open, shut = "(", ")"
		filled, rest, edge = st.Disabled, st.Disabled, st.Disabled
	case m.focused:
		open, shut = "<", ">"
		edge = st.Focus.Bold()
		filled = filled.Bold()
	}
	out := edge.Render(open) +
		filled.Render(strings.Repeat(g.BarFull, at)+g.Dot) +
		rest.Render(strings.Repeat(g.BarEmpty, m.width()-1-at)) +
		edge.Render(shut)
	if m.ShowValue {
		out += " " + edge.Render(m.text())
	}
	return out
}

// Compile-time proof that Model satisfies tui.Component[Model] — see
// tui.Component's doc comment for what this contract means and why.
var _ tui.Component[Model] = Model{}

// Linearize renders the slider as plain text for accessible output (see
// tui.Linearizer): "slider, 40, from 0 to 100", with ", unavailable" when
// disabled.
func (m Model) Linearize() string {
	f := func(v float64) string { return strconv.FormatFloat(v, 'f', -1, 64) }
	out := "slider, " + m.text() + ", from " + f(m.Min) + " to " + f(m.max())
	if m.Disabled {
		out += ", unavailable"
	}
	return out
}
