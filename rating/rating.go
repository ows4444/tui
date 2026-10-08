// Package rating is a score out of a small maximum, shown as a row of filled
// and empty marks: Left and Right lower and raise it, a digit sets it, and
// with the mouse on a click on a mark sets it to that mark.
//
// Stability: experimental. Its API may change in any minor release.
package rating

import (
	"strconv"
	"strings"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/hittest"
	"github.com/ows4444/tui/keymap"
	"github.com/ows4444/tui/theme"
)

// Model is one rating. The zero value is not ready to use; build one with
// New.
type Model struct {
	// ID is carried by ChangedMsg, to tell one rating's change from
	// another's.
	ID string
	// Max is the highest score, and the number of marks drawn. A Max under 1
	// is read as 5.
	Max int
	// ReadOnly shows the score and takes no input; it is drawn as an
	// enabled rating is.
	ReadOnly bool
	// Disabled takes no input and is drawn dimmed.
	Disabled bool
	// ShowValue draws the score after the marks, as "3/5".
	ShowValue bool
	Theme     theme.Theme

	// tokens is the per-instance colour override set by WithTokens.
	tokens theme.Tokens
	// KeyMap holds the keys for each action. New fills it with DefaultKeyMap;
	// a Model built as a struct literal with a zero KeyMap behaves as if it
	// held DefaultKeyMap.
	KeyMap KeyMap

	// Mouse, when true, makes Update handle tui.MouseEvent: a left press on
	// a mark sets the score to it, and a press on the mark the score is
	// already at clears the score. Off (the default) ignores the mouse.
	Mouse bool
	// Bounds is the screen rectangle where the app draws the rating. The
	// marks start one cell in from its left edge.
	Bounds hittest.Rect

	value   int
	focused bool
}

// New returns a rating out of max with a score of 0.
func New(max int) Model {
	return Model{Max: max, Theme: theme.DarkTheme(), KeyMap: DefaultKeyMap()}
}

// KeyMap names the keys of each action of a Model. A digit key always sets
// the score to that digit and is not in the KeyMap.
type KeyMap struct {
	Dec   keymap.Binding // lower the score by one
	Inc   keymap.Binding // raise the score by one
	Clear keymap.Binding // set the score to 0
	Full  keymap.Binding // set the score to Max
}

// DefaultKeyMap returns Left and Right, Home or Backspace, and End.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Dec:   keymap.NewBinding("lower", "left", "down"),
		Inc:   keymap.NewBinding("raise", "right", "up"),
		Clear: keymap.NewBinding("clear", "home", "backspace"),
		Full:  keymap.NewBinding("highest", "end"),
	}
}

func (m Model) keys() KeyMap {
	km := m.KeyMap
	if len(km.Dec.Keys)+len(km.Inc.Keys)+len(km.Clear.Keys)+len(km.Full.Keys) == 0 {
		return DefaultKeyMap()
	}
	return km
}

// Bindings returns the actions the rating honours, with descriptions, for
// help text. A read-only or disabled rating has none.
func (m Model) Bindings() []keymap.Binding {
	if !m.editable() {
		return nil
	}
	km := m.keys()
	return []keymap.Binding{km.Dec, km.Inc, km.Clear, km.Full}
}

// ChangedMsg is delivered (via the Cmd Update returns) when the score
// changes.
type ChangedMsg struct {
	// ID is the rating's ID.
	ID string
	// Value is the new score.
	Value int
}

func (m Model) max() int {
	if m.Max < 1 {
		return 5
	}
	return m.Max
}

func (m Model) editable() bool { return !m.ReadOnly && !m.Disabled }

// Value returns the score, from 0 to Max.
func (m Model) Value() int { return min(max(m.value, 0), m.max()) }

// SetValue sets the score, held between 0 and Max, without a key press: no
// ChangedMsg is delivered.
func (m *Model) SetValue(v int) { m.value = min(max(v, 0), m.max()) }

// Focus gives the rating keyboard focus. It returns no Cmd; the result is
// there so a rating can stand where any focusable widget does.
func (m *Model) Focus() tui.Cmd {
	m.focused = true
	return nil
}

// Blur takes keyboard focus away.
func (m *Model) Blur() { m.focused = false }

// Focused reports whether the rating has keyboard focus.
func (m Model) Focused() bool { return m.focused }

// set moves the score to v and returns the Cmd that reports it, or nil when
// the score did not change.
func (m Model) set(v int) (Model, tui.Cmd) {
	v = min(max(v, 0), m.max())
	if v == m.Value() {
		return m, nil
	}
	m.value = v
	msg := ChangedMsg{ID: m.ID, Value: v}
	return m, func() tui.Msg { return msg }
}

// Update lowers and raises the score on Left and Right, clears it on Home or
// Backspace, fills it on End, and sets it to a digit that is typed, when the
// rating has focus. With Mouse on, a left press on a mark sets the score to
// it; on the mark the score is at, it clears the score. A read-only or
// disabled rating ignores both.
func (m Model) Update(msg tui.Msg) (Model, tui.Cmd) {
	if ev, ok := msg.(tui.MouseEvent); ok {
		return m.updateMouse(ev)
	}
	if !m.focused || !m.editable() {
		return m, nil
	}
	km := m.keys()
	switch {
	case keymap.Matches(msg, km.Dec):
		return m.set(m.Value() - 1)
	case keymap.Matches(msg, km.Inc):
		return m.set(m.Value() + 1)
	case keymap.Matches(msg, km.Clear):
		return m.set(0)
	case keymap.Matches(msg, km.Full):
		return m.set(m.max())
	}
	if k, ok := msg.(tui.Key); ok && k.Type == tui.KeyRunes && len(k.Text) == 1 && k.Text[0] >= '0' && k.Text[0] <= '9' {
		return m.set(int(k.Text[0] - '0'))
	}
	return m, nil
}

func (m Model) updateMouse(ev tui.MouseEvent) (Model, tui.Cmd) {
	if !m.Mouse || !m.editable() || ev.Action != tui.MouseActionPress || ev.Button != tui.MouseButtonLeft ||
		!m.Bounds.Contains(ev.X, ev.Y) {
		return m, nil
	}
	mark := ev.X - m.Bounds.X // the marks start after the left bracket cell
	if mark < 1 || mark > m.max() {
		return m, nil
	}
	if mark == m.Value() {
		return m.set(0)
	}
	return m.set(mark)
}

// View renders Max marks on one row, filled up to the score and empty after
// it, between two cells that hold brackets when the rating has focus and
// blanks when it does not, so focus reads without colour and does not change
// the width. A disabled rating is dimmed and held in parentheses. ShowValue
// adds the score, as "3/5", after a space.
func (m Model) View() string {
	t := m.themed()
	st := t.ResolvedStates()
	g := t.GlyphSet()
	on := ansi.NewStyle().Foreground(t.Warning)
	off := ansi.NewStyle().Foreground(t.Muted)
	open, shut := " ", " "
	edge := ansi.NewStyle().Foreground(t.Text)
	switch {
	case m.Disabled:
		open, shut = "(", ")"
		on, off, edge = st.Disabled, st.Disabled, st.Disabled
	case m.focused:
		open, shut = "[", "]"
		edge = st.Focus.Bold()
		on = on.Bold()
	}
	v := m.Value()
	out := edge.Render(open) +
		on.Render(strings.Repeat(g.Dot, v)) +
		off.Render(strings.Repeat(g.DotEmpty, m.max()-v)) +
		edge.Render(shut)
	if m.ShowValue {
		out += " " + edge.Render(strconv.Itoa(v)+"/"+strconv.Itoa(m.max()))
	}
	return out
}

// Compile-time proof that Model satisfies tui.Component[Model] — see
// tui.Component's doc comment for what this contract means and why.
var _ tui.Component[Model] = Model{}

// Linearize renders the rating as plain text for accessible output (see
// tui.Linearizer): "rating, 3 of 5", with ", read only" or ", unavailable".
func (m Model) Linearize() string {
	out := "rating, " + strconv.Itoa(m.Value()) + " of " + strconv.Itoa(m.max())
	switch {
	case m.Disabled:
		out += ", unavailable"
	case m.ReadOnly:
		out += ", read only"
	}
	return out
}
