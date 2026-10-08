// Package checkbox is one box that is checked or not, with a label: Space
// checks and unchecks it when it has focus, and with the mouse on so does a
// click. For a list of them with one cursor, see multiselect.
//
// Stability: experimental. Its API may change in any minor release.
package checkbox

import (
	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/hittest"
	"github.com/ows4444/tui/keymap"
	"github.com/ows4444/tui/theme"
)

// Model is one checkbox. The zero value is not ready to use; build one with
// New.
type Model struct {
	// ID is carried by ChangedMsg, to tell one checkbox's change from
	// another's.
	ID string
	// Label is the text after the box. An empty Label draws the box alone.
	Label string
	// Indeterminate draws the box as "[-]": neither checked nor unchecked,
	// as the box over a list some of whose items are checked. The next press
	// checks it and clears Indeterminate.
	Indeterminate bool
	// Disabled stops the box being changed; it is drawn faint.
	Disabled bool
	Theme    theme.Theme

	// tokens is the per-instance colour override set by WithTokens.
	tokens theme.Tokens
	// KeyMap holds the key that checks and unchecks the box. New fills it
	// with DefaultKeyMap; a Model built as a struct literal with a zero
	// KeyMap behaves as if it held DefaultKeyMap.
	KeyMap KeyMap

	// Mouse, when true, makes Update handle tui.MouseEvent: a left press on
	// the box or its label checks or unchecks it. Off (the default) ignores
	// the mouse.
	Mouse bool
	// Bounds is the screen rectangle where the app draws the checkbox.
	Bounds hittest.Rect

	checked bool
	focused bool
}

// New returns an unchecked checkbox labelled label.
func New(label string) Model {
	return Model{Label: label, Theme: theme.DarkTheme(), KeyMap: DefaultKeyMap()}
}

// KeyMap names the key of a Model's one action.
type KeyMap struct {
	Toggle keymap.Binding // check or uncheck the box
}

// DefaultKeyMap returns Space. Enter is left out, since in a form it moves
// to the next field.
func DefaultKeyMap() KeyMap {
	return KeyMap{Toggle: keymap.NewBinding("check", "space")}
}

func (m Model) keys() KeyMap {
	if len(m.KeyMap.Toggle.Keys) == 0 {
		return DefaultKeyMap()
	}
	return m.KeyMap
}

// Bindings returns the action the checkbox honours, for help text. A
// disabled one has none.
func (m Model) Bindings() []keymap.Binding {
	if m.Disabled {
		return nil
	}
	return []keymap.Binding{m.keys().Toggle}
}

// ChangedMsg is delivered (via the Cmd Update returns) when a key or a click
// checks or unchecks the box.
type ChangedMsg struct {
	// ID is the checkbox's ID.
	ID string
	// Checked is the box's state after the change.
	Checked bool
}

// Checked reports whether the box is checked. An indeterminate box is not.
func (m Model) Checked() bool { return m.checked && !m.Indeterminate }

// SetChecked checks or unchecks the box without a key press: no ChangedMsg
// is delivered. It clears Indeterminate.
func (m *Model) SetChecked(checked bool) { m.checked, m.Indeterminate = checked, false }

// Focus gives the checkbox keyboard focus. It returns no Cmd; the result is
// there so a checkbox can stand where any focusable widget does.
func (m *Model) Focus() tui.Cmd {
	m.focused = true
	return nil
}

// Blur takes keyboard focus away.
func (m *Model) Blur() { m.focused = false }

// Focused reports whether the checkbox has keyboard focus.
func (m Model) Focused() bool { return m.focused }

// toggle flips the box and returns the Cmd that reports it. An
// indeterminate box becomes checked.
func (m Model) toggle() (Model, tui.Cmd) {
	m.checked = m.Indeterminate || !m.checked
	m.Indeterminate = false
	msg := ChangedMsg{ID: m.ID, Checked: m.checked}
	return m, func() tui.Msg { return msg }
}

// Update checks or unchecks the box on Space when it has focus, and, with
// Mouse on, on a left press inside Bounds. A disabled checkbox ignores both.
func (m Model) Update(msg tui.Msg) (Model, tui.Cmd) {
	if m.Disabled {
		return m, nil
	}
	if ev, ok := msg.(tui.MouseEvent); ok {
		if m.Mouse && ev.Action == tui.MouseActionPress && ev.Button == tui.MouseButtonLeft &&
			m.Bounds.Contains(ev.X, ev.Y) {
			return m.toggle()
		}
		return m, nil
	}
	if m.focused && keymap.Matches(msg, m.keys().Toggle) {
		return m.toggle()
	}
	return m, nil
}

func (m Model) label() string { return ansi.Sanitize(m.Label) }

// View renders "[x] Label", "[ ] Label" or, when Indeterminate, "[-] Label",
// so the state reads without colour. A checkbox that has focus is drawn with
// angle brackets, "<x> Label", in the focus style; a disabled one is faint.
func (m Model) View() string {
	t := m.themed()
	st := t.ResolvedStates()
	mark := " "
	box := ansi.NewStyle().Foreground(t.Muted)
	switch {
	case m.Indeterminate:
		mark = "-"
		box = ansi.NewStyle().Foreground(t.Primary)
	case m.checked:
		mark = "x"
		box = ansi.NewStyle().Foreground(t.Success)
	}
	open, shut := "[", "]"
	text := ansi.NewStyle().Foreground(t.Text)
	switch {
	case m.Disabled:
		box, text = st.Disabled.Faint(), st.Disabled.Faint()
	case m.focused:
		open, shut = "<", ">"
		box = st.Focus.Bold()
	}
	out := box.Render(open + mark + shut)
	if l := m.label(); l != "" {
		out += " " + text.Render(l)
	}
	return out
}

// Width is the number of cells View takes.
func (m Model) Width() int {
	if l := m.label(); l != "" {
		return 4 + ansi.Width(l)
	}
	return 3
}

// Compile-time proof that Model satisfies tui.Component[Model] — see
// tui.Component's doc comment for what this contract means and why.
var _ tui.Component[Model] = Model{}

// Linearize renders the checkbox as plain text for accessible output (see
// tui.Linearizer): "Keep metadata, checkbox, checked", with "not checked" or
// "partly checked", and ", unavailable" when disabled.
func (m Model) Linearize() string {
	state := "not checked"
	switch {
	case m.Indeterminate:
		state = "partly checked"
	case m.checked:
		state = "checked"
	}
	out := "checkbox, " + state
	if l := m.label(); l != "" {
		out = l + ", " + out
	}
	if m.Disabled {
		out += ", unavailable"
	}
	return out
}
