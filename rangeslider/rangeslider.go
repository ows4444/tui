// Package rangeslider is a range chosen from a span by moving two thumbs along
// a track: a low and a high end, the part between them filled. The active thumb
// is moved with Left and Right (a step) or Page Up and Page Down (a larger
// one), Space switches thumbs, and "[" and "]" pick the low or the high one.
// With the mouse on, a press takes the thumb nearest the pointer and puts it
// under the pointer, and it follows a drag. The low thumb never passes the high
// one.
//
// Stability: experimental. Its API may change in any minor release.
package rangeslider

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

// Model is one range slider. Build one with New.
type Model struct {
	// ID is carried by ChangedMsg, to tell one slider's change from another's.
	ID string
	// Min and Max are the ends of the span. A Max below Min is read as Min.
	Min, Max float64
	// Step is how far Left and Right move a thumb, and what the values are
	// rounded to. A Step of 0 or less is read as 1.
	Step float64
	// BigStep is how far Page Up and Page Down move a thumb. A BigStep of 0 or
	// less is read as ten Steps.
	BigStep float64
	// Width is the length of the track in cells. A Width under 3 is read as 3.
	Width int
	// ShowValue draws "low to high" after the track.
	ShowValue bool
	// Format turns a value into text for ShowValue and Linearize. Nil prints
	// the shortest exact decimal.
	Format func(float64) string
	// Disabled stops the values being changed; the slider is drawn dimmed.
	Disabled bool
	Theme    theme.Theme

	tokens theme.Tokens
	// KeyMap holds the keys for each action. New fills it with DefaultKeyMap.
	KeyMap KeyMap

	// Mouse, when true, makes Update handle tui.MouseEvent: see the package
	// comment. Off (the default) ignores the mouse.
	Mouse bool
	// Bounds is the screen rectangle where the app draws the slider. The track
	// starts one cell in from its left edge.
	Bounds hittest.Rect

	lo, hi   float64
	active   int // 0 the low thumb, 1 the high one
	focused  bool
	dragging bool
}

// New returns a range slider over min to max with a Step of 1, a track of 24
// cells, and the whole span selected.
func New(min, max float64) Model {
	m := Model{Min: min, Max: max, Step: 1, Width: 24, Theme: theme.DarkTheme(), KeyMap: DefaultKeyMap()}
	m.lo, m.hi = min, m.span()
	return m
}

// KeyMap names the keys of each action of a Model.
type KeyMap struct {
	Dec    keymap.Binding // lower the active thumb by Step
	Inc    keymap.Binding // raise the active thumb by Step
	DecBig keymap.Binding // lower it by BigStep
	IncBig keymap.Binding // raise it by BigStep
	ToMin  keymap.Binding // the active thumb to its lowest
	ToMax  keymap.Binding // the active thumb to its highest
	Switch keymap.Binding // make the other thumb the active one
	Low    keymap.Binding // make the low thumb the active one
	High   keymap.Binding // make the high thumb the active one
}

// DefaultKeyMap returns the arrow keys, Page Up and Page Down, Home and End,
// Space to switch thumbs and "[" and "]" to pick one.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Dec:    keymap.NewBinding("lower", "left", "down"),
		Inc:    keymap.NewBinding("raise", "right", "up"),
		DecBig: keymap.NewBinding("lower more", "pgdown"),
		IncBig: keymap.NewBinding("raise more", "pgup"),
		ToMin:  keymap.NewBinding("lowest", "home"),
		ToMax:  keymap.NewBinding("highest", "end"),
		Switch: keymap.NewBinding("other thumb", "space", " "),
		Low:    keymap.NewBinding("low thumb", "["),
		High:   keymap.NewBinding("high thumb", "]"),
	}
}

func (km KeyMap) list() []keymap.Binding {
	return []keymap.Binding{km.Dec, km.Inc, km.DecBig, km.IncBig, km.ToMin, km.ToMax, km.Switch, km.Low, km.High}
}

func (m Model) keys() KeyMap {
	for _, b := range m.KeyMap.list() {
		if len(b.Keys) > 0 {
			return m.KeyMap
		}
	}
	return DefaultKeyMap()
}

// Bindings returns the actions the slider honours, for help text. A disabled
// slider has none.
func (m Model) Bindings() []keymap.Binding {
	if m.Disabled {
		return nil
	}
	return m.keys().list()
}

// ChangedMsg is delivered (via the Cmd Update returns) when either end changes.
type ChangedMsg struct {
	// ID is the slider's ID.
	ID string
	// Low and High are the new ends.
	Low, High float64
}

func (m Model) span() float64 { return math.Max(m.Min, m.Max) }
func (m Model) step() float64 { return positive(m.Step, 1) }
func (m Model) big() float64  { return positive(m.BigStep, 10*m.step()) }
func (m Model) width() int    { return max(m.Width, 3) }

func positive(v, fallback float64) float64 {
	if v > 0 {
		return v
	}
	return fallback
}

// snap holds v inside the span and rounds it to a whole number of Steps from
// Min. Max is always reachable.
func (m Model) snap(v float64) float64 {
	if math.IsNaN(v) || v <= m.Min {
		return m.Min
	}
	if v >= m.span() {
		return m.span()
	}
	s := m.step()
	return math.Min(m.Min+math.Round((v-m.Min)/s)*s, m.span())
}

// Low returns the low end of the range.
func (m Model) Low() float64 { return m.snap(m.lo) }

// High returns the high end of the range.
func (m Model) High() float64 { return math.Max(m.snap(m.hi), m.Low()) }

// SetRange sets both ends, held inside the span and rounded to a Step, the low
// end swapped with the high one if it is above it. No ChangedMsg is delivered.
func (m *Model) SetRange(low, high float64) {
	lo, hi := m.snap(low), m.snap(high)
	if lo > hi {
		lo, hi = hi, lo
	}
	m.lo, m.hi = lo, hi
}

// Focus gives the slider keyboard focus. It returns no Cmd.
func (m *Model) Focus() tui.Cmd { m.focused = true; return nil }

// Blur takes keyboard focus away, and lets go of a drag in progress.
func (m *Model) Blur() { m.focused, m.dragging = false, false }

// Focused reports whether the slider has keyboard focus.
func (m Model) Focused() bool { return m.focused }

// Active is the thumb the keys move: 0 for the low one, 1 for the high one.
func (m Model) Active() int { return m.active }

// setActive moves the active thumb to v, not past the other, and returns the
// Cmd that reports a change.
func (m Model) setActive(v float64) (Model, tui.Cmd) {
	v = m.snap(v)
	lo, hi := m.Low(), m.High()
	if m.active == 0 {
		v = math.Min(v, hi)
		if v == lo {
			return m, nil
		}
		m.lo, m.hi = v, hi
	} else {
		v = math.Max(v, lo)
		if v == hi {
			return m, nil
		}
		m.lo, m.hi = lo, v
	}
	msg := ChangedMsg{ID: m.ID, Low: m.Low(), High: m.High()}
	return m, func() tui.Msg { return msg }
}

func (m Model) activeValue() float64 {
	if m.active == 0 {
		return m.Low()
	}
	return m.High()
}

// Update moves the active thumb on the arrow keys, Page Up and Page Down, Home
// and End when the slider has focus, and with Mouse on, on a press or a drag.
func (m Model) Update(msg tui.Msg) (Model, tui.Cmd) {
	if ev, ok := msg.(tui.MouseEvent); ok {
		return m.updateMouse(ev)
	}
	if !m.focused || m.Disabled {
		return m, nil
	}
	km := m.keys()
	v := m.activeValue()
	switch {
	case keymap.Matches(msg, km.Dec):
		return m.setActive(v - m.step())
	case keymap.Matches(msg, km.Inc):
		return m.setActive(v + m.step())
	case keymap.Matches(msg, km.DecBig):
		return m.setActive(v - m.big())
	case keymap.Matches(msg, km.IncBig):
		return m.setActive(v + m.big())
	case keymap.Matches(msg, km.ToMin):
		return m.setActive(m.Min)
	case keymap.Matches(msg, km.ToMax):
		return m.setActive(m.span())
	case keymap.Matches(msg, km.Switch):
		m.active = 1 - m.active
	case keymap.Matches(msg, km.Low):
		m.active = 0
	case keymap.Matches(msg, km.High):
		m.active = 1
	}
	return m, nil
}

// valueAt is the value of track cell x, where 0 is the first cell.
func (m Model) valueAt(x int) float64 {
	last := m.width() - 1
	x = min(max(x, 0), last)
	return m.Min + (m.span()-m.Min)*float64(x)/float64(last)
}

// cell is the track cell a value sits on.
func (m Model) cell(v float64) int {
	span := m.span() - m.Min
	if span <= 0 {
		return 0
	}
	return int(math.Round((v - m.Min) / span * float64(m.width()-1)))
}

func (m Model) updateMouse(ev tui.MouseEvent) (Model, tui.Cmd) {
	if !m.Mouse || m.Disabled {
		return m, nil
	}
	x := ev.X - m.Bounds.X - 1
	switch ev.Action {
	case tui.MouseActionPress:
		if ev.Button != tui.MouseButtonLeft || !m.Bounds.Contains(ev.X, ev.Y) {
			return m, nil
		}
		// The nearest thumb takes the press; at a tie the one the press is
		// on the far side of, so two thumbs on one cell can be pulled apart.
		dl, dh := abs(x-m.cell(m.Low())), abs(x-m.cell(m.High()))
		switch {
		case dl < dh:
			m.active = 0
		case dh < dl:
			m.active = 1
		case x < m.cell(m.Low()):
			m.active = 0
		default:
			m.active = 1
		}
		m.dragging = true
		return m.setActive(m.valueAt(x))
	case tui.MouseActionMotion:
		if m.dragging {
			return m.setActive(m.valueAt(x))
		}
	case tui.MouseActionRelease:
		m.dragging = false
	}
	return m, nil
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func (m Model) text(v float64) string {
	if m.Format != nil {
		return ansi.Sanitize(m.Format(v))
	}
	return strconv.FormatFloat(v, 'f', -1, 64)
}

// View renders the track on one row: empty before the low thumb, filled between
// the thumbs, empty after the high one, a dot for each thumb, between two cells
// that hold angle brackets when the slider has focus and blanks when it does
// not. The active thumb is bold while the slider has focus. A disabled slider
// is dimmed and held in parentheses. ShowValue adds "low to high" after a space.
func (m Model) View() string {
	t := m.themed()
	st := t.ResolvedStates()
	g := t.GlyphSet()
	a, b := m.cell(m.Low()), m.cell(m.High())
	out := edge(m, st, t)
	filled := ansi.NewStyle().Foreground(t.Primary)
	rest := ansi.NewStyle().Foreground(t.Muted)
	thumbLo, thumbHi := filled, filled
	open, shut := " ", " "
	switch {
	case m.Disabled:
		open, shut = "(", ")"
		filled, rest, thumbLo, thumbHi = st.Disabled, st.Disabled, st.Disabled, st.Disabled
	case m.focused:
		open, shut = "<", ">"
		if m.active == 0 {
			thumbLo = st.Focus.Bold()
		} else {
			thumbHi = st.Focus.Bold()
		}
	}
	var track strings.Builder
	for i := 0; i < m.width(); i++ {
		switch {
		case i == a && i == b:
			if m.active == 0 {
				track.WriteString(thumbLo.Render(g.Dot))
			} else {
				track.WriteString(thumbHi.Render(g.Dot))
			}
		case i == a:
			track.WriteString(thumbLo.Render(g.Dot))
		case i == b:
			track.WriteString(thumbHi.Render(g.Dot))
		case i > a && i < b:
			track.WriteString(filled.Render(g.BarFull))
		default:
			track.WriteString(rest.Render(g.BarEmpty))
		}
	}
	s := out.Render(open) + track.String() + out.Render(shut)
	if m.ShowValue {
		s += " " + out.Render(m.text(m.Low())+" to "+m.text(m.High()))
	}
	return s
}

// edge is the style of the bracket cells and the value text.
func edge(m Model, st theme.States, t theme.Theme) ansi.Style {
	switch {
	case m.Disabled:
		return st.Disabled
	case m.focused:
		return st.Focus.Bold()
	}
	return ansi.NewStyle().Foreground(t.Text)
}

// Compile-time proof that Model satisfies tui.Component[Model].
var _ tui.Component[Model] = Model{}

// Linearize renders the slider as plain text for accessible output (see
// tui.Linearizer): "range slider, 20 to 60, from 0 to 100", with ", unavailable"
// when disabled.
func (m Model) Linearize() string {
	out := "range slider, " + m.text(m.Low()) + " to " + m.text(m.High()) +
		", from " + m.text(m.Min) + " to " + m.text(m.span())
	if m.Disabled {
		out += ", unavailable"
	}
	return out
}
