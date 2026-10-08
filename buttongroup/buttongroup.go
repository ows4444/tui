// Package buttongroup is a row of buttons that take one place in the focus
// order: Left and Right move between them, and Enter or Space presses the
// one the cursor is on.
//
// In ModeActions the buttons are plain and each press is an action. In
// ModeSingle and ModeMultiple they are toggle buttons and the group is a
// choice: one of them, or any number.
//
// Stability: experimental. Its API may change in any minor release.
package buttongroup

import (
	"strconv"
	"strings"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/button"
	"github.com/ows4444/tui/hittest"
	"github.com/ows4444/tui/keymap"
	"github.com/ows4444/tui/theme"
)

// Mode is what the buttons of a group are for.
type Mode int

const (
	// ModeActions is a row of plain buttons; a press is an action.
	ModeActions Mode = iota
	// ModeSingle is a row of toggle buttons of which at most one is on.
	ModeSingle
	// ModeMultiple is a row of toggle buttons, each on or off by itself.
	ModeMultiple
)

// Model is a row of buttons with a cursor. The zero value is an empty
// ModeActions group; build one with New.
type Model struct {
	// ID is carried by ChangedMsg, to tell one group's change from another's.
	ID string
	// Buttons are the group's buttons, left to right. Each keeps its own
	// Label, ID, Variant, Size and Disabled; the group sets their focus,
	// their toggle state in ModeSingle, and their Mouse and Bounds.
	Buttons []button.Model
	Mode    Mode
	// Required, in ModeSingle, stops the button that is on being turned off
	// by pressing it again, so that once a choice is made one is always on.
	Required bool
	// Gap is the number of blank cells between two buttons.
	Gap int

	// KeyMap holds the keys that move the cursor. New fills it with
	// DefaultKeyMap; a Model built as a struct literal with a zero KeyMap
	// behaves as if it held DefaultKeyMap.
	KeyMap KeyMap

	// Mouse, when true, makes Update handle tui.MouseEvent: a click on a
	// button presses it and moves the cursor to it. Off (the default)
	// ignores the mouse.
	Mouse bool
	// Bounds is the screen rectangle where the app draws the group; its
	// left edge is the first button's.
	Bounds hittest.Rect

	focused bool
	cursor  int
}

// New returns a group in the given mode with one button per label, a gap of
// one cell, and the cursor on the first button. Each button's ID is its
// label. In ModeSingle and ModeMultiple the buttons are toggle buttons.
func New(mode Mode, labels ...string) Model {
	m := Model{Mode: mode, Gap: 1, KeyMap: DefaultKeyMap()}
	for _, l := range labels {
		b := button.New(l)
		b.ID = l
		b.Toggle = mode != ModeActions
		m.Buttons = append(m.Buttons, b)
	}
	return m
}

// KeyMap names the keys that move a group's cursor.
type KeyMap struct {
	Prev  keymap.Binding // move to the button before
	Next  keymap.Binding // move to the button after
	First keymap.Binding // move to the first button
	Last  keymap.Binding // move to the last button
}

// DefaultKeyMap returns Left, Right, Home and End.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Prev:  keymap.NewBinding("previous", "left"),
		Next:  keymap.NewBinding("next", "right"),
		First: keymap.NewBinding("first", "home"),
		Last:  keymap.NewBinding("last", "end"),
	}
}

func (m Model) keys() KeyMap {
	km := m.KeyMap
	if len(km.Prev.Keys)+len(km.Next.Keys)+len(km.First.Keys)+len(km.Last.Keys) == 0 {
		return DefaultKeyMap()
	}
	return km
}

// Bindings returns the keys the group honours, for help text: its own, then
// those of the button under the cursor.
func (m Model) Bindings() []keymap.Binding {
	km := m.keys()
	out := []keymap.Binding{km.Prev, km.Next, km.First, km.Last}
	if b, ok := m.current(); ok {
		out = append(out, b.Bindings()...)
	}
	return out
}

// ChangedMsg is delivered (via the Cmd Update returns) when a press changed
// which buttons of a ModeSingle or ModeMultiple group are on. It stands in
// place of the pressed button's own button.PressedMsg, so one press is one
// message. A ModeActions group delivers button.PressedMsg.
type ChangedMsg struct {
	// ID is the group's ID.
	ID string
	// Pressed is the ID of the button whose press made the change.
	Pressed string
	// On holds the IDs of the buttons that are on, left to right.
	On []string
}

// usable reports whether the cursor may rest on button i.
func (m Model) usable(i int) bool {
	return i >= 0 && i < len(m.Buttons) && !m.Buttons[i].Disabled
}

// current returns the button under the cursor, if the cursor is on one.
func (m Model) current() (button.Model, bool) {
	if !m.usable(m.cursor) {
		return button.Model{}, false
	}
	return m.Buttons[m.cursor], true
}

// sync gives focus to the button under the cursor when the group has it and
// takes it from every other. Buttons is copied first: a Model is a value, and
// a copy of it must not change through the slice it shares.
func (m Model) sync() Model {
	if !m.usable(m.cursor) {
		m.cursor = m.step(-1, 1)
	}
	bs := make([]button.Model, len(m.Buttons))
	copy(bs, m.Buttons)
	for i := range bs {
		switch {
		case m.focused && i == m.cursor && !bs[i].Focused():
			bs[i].Focus()
		case (!m.focused || i != m.cursor) && bs[i].Focused():
			bs[i].Blur()
		}
	}
	m.Buttons = bs
	return m
}

// step returns the first usable button from, and not counting, index from in
// direction dir, wrapping round; from itself when there is no other.
func (m Model) step(from, dir int) int {
	n := len(m.Buttons)
	for k := 1; k <= n; k++ {
		if i := ((from+dir*k)%n + n) % n; m.usable(i) {
			return i
		}
	}
	return from
}

// Focus gives the group keyboard focus: the button under the cursor shows
// it, in angle brackets. It returns no Cmd; the result is there so a group can stand where any
// focusable widget does.
func (m *Model) Focus() tui.Cmd {
	m.focused = true
	*m = m.sync()
	return nil
}

// Blur takes keyboard focus away.
func (m *Model) Blur() {
	m.focused = false
	*m = m.sync()
}

// Focused reports whether the group has keyboard focus.
func (m Model) Focused() bool { return m.focused }

// Cursor returns the index of the button the cursor is on.
func (m Model) Cursor() int { return m.cursor }

// SetCursor moves the cursor to button i. An index that is out of range, or
// names a disabled button, is ignored.
func (m *Model) SetCursor(i int) {
	if m.usable(i) {
		m.cursor = i
		*m = m.sync()
	}
}

// On returns the IDs of the buttons that are on, left to right. It is empty
// in ModeActions.
func (m Model) On() []string {
	var out []string
	for _, b := range m.Buttons {
		if b.On() {
			out = append(out, b.ID)
		}
	}
	return out
}

// SetOn turns on the buttons with the given IDs and turns the others off,
// without a press: no message is delivered. In ModeSingle only the first of
// ids that names a button is turned on.
func (m *Model) SetOn(ids ...string) {
	bs := make([]button.Model, len(m.Buttons))
	copy(bs, m.Buttons)
	done := false
	for i := range bs {
		on := false
		for _, id := range ids {
			on = on || bs[i].ID == id
		}
		if m.Mode == ModeSingle && done {
			on = false
		}
		bs[i].SetOn(on)
		done = done || bs[i].On()
	}
	m.Buttons = bs
}

// Update moves the cursor on Left, Right, Home and End, skipping disabled
// buttons and wrapping at the ends, and passes every other message to the
// button under the cursor, so Enter or Space presses it. With Mouse on, a
// click on a button moves the cursor to it and presses it. Only a group that
// has focus acts on keys.
func (m Model) Update(msg tui.Msg) (Model, tui.Cmd) {
	if ev, ok := msg.(tui.MouseEvent); ok {
		return m.updateMouse(ev)
	}
	if !m.focused || len(m.Buttons) == 0 {
		return m, nil
	}
	km := m.keys()
	switch {
	case keymap.Matches(msg, km.Prev):
		m.cursor = m.step(m.cursor, -1)
		return m.sync(), nil
	case keymap.Matches(msg, km.Next):
		m.cursor = m.step(m.cursor, 1)
		return m.sync(), nil
	case keymap.Matches(msg, km.First):
		m.cursor = m.step(-1, 1)
		return m.sync(), nil
	case keymap.Matches(msg, km.Last):
		m.cursor = m.step(len(m.Buttons), -1)
		return m.sync(), nil
	}
	m = m.sync()
	if !m.usable(m.cursor) {
		return m, nil
	}
	return m.forward(m.cursor, msg)
}

// forward passes msg to button i and settles the group's choice after it.
func (m Model) forward(i int, msg tui.Msg) (Model, tui.Cmd) {
	before := m.Buttons[i].On()
	if m.Mode == ModeSingle && m.Required && before {
		// The press would turn off the only button that is on. Let the
		// button see the message for its pointer state, and put it back on.
		b, _ := m.Buttons[i].Update(msg)
		b.SetOn(true)
		m.Buttons[i] = b
		return m, nil
	}
	b, cmd := m.Buttons[i].Update(msg)
	m.Buttons[i] = b
	if b.On() == before {
		return m, cmd
	}
	if m.Mode == ModeSingle && b.On() {
		for j := range m.Buttons {
			if j != i {
				m.Buttons[j].SetOn(false)
			}
		}
	}
	id, pressed, on := m.ID, b.ID, m.On()
	return m, func() tui.Msg { return ChangedMsg{ID: id, Pressed: pressed, On: on} }
}

// rects returns the screen rectangle of each button.
func (m Model) rects() []hittest.Rect {
	out := make([]hittest.Rect, len(m.Buttons))
	x := m.Bounds.X
	for i, b := range m.Buttons {
		out[i] = hittest.Rect{X: x, Y: m.Bounds.Y, W: b.Width(), H: 1}
		x += b.Width() + m.gap()
	}
	return out
}

func (m Model) gap() int {
	if m.Gap < 0 {
		return 0
	}
	return m.Gap
}

// updateMouse gives the event to every button, each with its own rectangle,
// so each keeps its hover and held-down state; a press on a button moves the
// cursor to it.
func (m Model) updateMouse(ev tui.MouseEvent) (Model, tui.Cmd) {
	if !m.Mouse {
		return m, nil
	}
	m = m.sync()
	var fired tui.Cmd // one event presses at most one button
	for i, r := range m.rects() {
		m.Buttons[i].Mouse, m.Buttons[i].Bounds = true, r
		if ev.Action == tui.MouseActionPress && ev.Button == tui.MouseButtonLeft &&
			r.Contains(ev.X, ev.Y) && m.usable(i) {
			m.cursor = i
		}
		var cmd tui.Cmd
		if m, cmd = m.forward(i, ev); cmd != nil {
			fired = cmd
		}
	}
	return m.sync(), fired
}

// View renders the buttons on one row, Gap cells apart.
func (m Model) View() string {
	parts := make([]string, len(m.Buttons))
	for i, b := range m.Buttons {
		parts[i] = b.View()
	}
	return strings.Join(parts, strings.Repeat(" ", m.gap()))
}

// Width is the number of cells View takes.
func (m Model) Width() int {
	w := 0
	for i, b := range m.Buttons {
		if i > 0 {
			w += m.gap()
		}
		w += b.Width()
	}
	return w
}

// Compile-time proof that Model satisfies tui.Component[Model] — see
// tui.Component's doc comment for what this contract means and why.
var _ tui.Component[Model] = Model{}

// SetTheme returns m with t applied to every button. It makes Model a
// tui.ThemeSetter, so a root model can forward the Program's theme (see
// tui.WithTheme). The group draws nothing of its own but the gaps.
func (m Model) SetTheme(t theme.Theme) Model {
	bs := make([]button.Model, len(m.Buttons))
	for i, b := range m.Buttons {
		bs[i] = b.SetTheme(t)
	}
	m.Buttons = bs
	return m
}

// Compile-time proof that Model satisfies tui.ThemeSetter[Model].
var _ tui.ThemeSetter[Model] = Model{}

// Linearize renders the group as plain text for accessible output (see
// tui.Linearizer): one line per button, as the button reads itself, with its
// position, e.g. "Bold, toggle button, on, 1 of 3".
func (m Model) Linearize() string {
	lines := make([]string, len(m.Buttons))
	for i, b := range m.Buttons {
		lines[i] = b.Linearize() + ", " + strconv.Itoa(i+1) + " of " + strconv.Itoa(len(m.Buttons))
	}
	return strings.Join(lines, "\n")
}
