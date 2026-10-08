// Package button is a pressable button: a label that takes focus and
// reports a press. Enter or Space presses it when focused, and so does a
// left click that goes down and comes up inside it.
//
// A button draws on one row. Its states are told apart without colour:
// square brackets mark it as a button, angle brackets mark focus ("< Save >"),
// reverse video marks a press, and a disabled or loading button says so in
// its label. Angle brackets are how every control of this family shows
// focus: buttongroup, radiogroup, checkbox, toggle, slider and rating too.
//
// With Toggle set it is a toggle button: each press turns it on or off, and
// a dot before the label shows that it is on.
//
// Stability: experimental. Its API may change in any minor release.
package button

import (
	"strings"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/hittest"
	"github.com/ows4444/tui/keymap"
	"github.com/ows4444/tui/theme"
)

// Variant is how much weight a button carries on the screen.
type Variant int

const (
	// VariantDefault is the main action: "[ Save ]" in the primary colour.
	VariantDefault Variant = iota
	// VariantSecondary is an action of less weight, in the text colour.
	VariantSecondary
	// VariantDestructive is an action that removes or cannot be undone, in
	// the error colour.
	VariantDestructive
	// VariantGhost has no brackets until it is focused or pressed.
	VariantGhost
	// VariantLink is drawn as underlined text with no brackets.
	VariantLink
)

// Size is how much room a button takes around its label.
type Size int

const (
	// SizeDefault pads the label with one space each side: "[ Save ]".
	SizeDefault Size = iota
	// SizeSmall has no padding: "[Save]".
	SizeSmall
	// SizeLarge pads the label with three spaces each side.
	SizeLarge
)

// Model is one button. The zero value is not ready to use; build one with
// New.
type Model struct {
	// ID is carried by PressedMsg, to tell one button's press from another's.
	ID string
	// Label is the button's text.
	Label   string
	Variant Variant
	Size    Size
	// Disabled stops the button being pressed; it is drawn dimmed.
	Disabled bool
	// Loading stops the button being pressed while the action it started is
	// still running, and adds an ellipsis to the label.
	Loading bool
	// Toggle makes this a toggle button: a press turns it on or off (see On)
	// as well as delivering PressedMsg. One cell before the label is kept
	// for the dot that marks it on, so turning it on does not change its
	// width.
	Toggle bool
	Theme  theme.Theme

	// tokens is the per-instance colour override set by WithTokens.
	tokens theme.Tokens
	// KeyMap holds the keys that press the button. New fills it with
	// DefaultKeyMap; a Model built as a struct literal with a zero KeyMap
	// behaves as if it held DefaultKeyMap.
	KeyMap KeyMap

	// Mouse, when true, makes Update handle tui.MouseEvent: a left button
	// that goes down and comes up inside Bounds presses the button, and
	// pointer motion over it sets Hovered. Off (the default) ignores the
	// mouse.
	Mouse bool
	// Bounds is the screen rectangle where the app draws the button.
	Bounds hittest.Rect

	focused bool
	hovered bool
	down    bool // the left button went down inside Bounds and is still held
	on      bool // a toggle button's state
}

// New returns a default-variant, default-size button labelled label.
func New(label string) Model {
	return Model{Label: label, Theme: theme.DarkTheme(), KeyMap: DefaultKeyMap()}
}

// KeyMap names the keys of a Model's one action.
type KeyMap struct {
	Press keymap.Binding // press the button
}

// DefaultKeyMap returns Enter and Space.
func DefaultKeyMap() KeyMap {
	return KeyMap{Press: keymap.NewBinding("press", "enter", "space")}
}

func (m Model) keys() KeyMap {
	if len(m.KeyMap.Press.Keys) == 0 {
		return DefaultKeyMap()
	}
	return m.KeyMap
}

// Bindings returns the action the button honours, for help text. A button
// that cannot be pressed has none.
func (m Model) Bindings() []keymap.Binding {
	if !m.pressable() {
		return nil
	}
	return []keymap.Binding{m.keys().Press}
}

// PressedMsg is delivered (via the Cmd Update returns) when the button is
// pressed, by key or by pointer.
type PressedMsg struct {
	// ID is the pressed button's ID.
	ID string
	// On is a toggle button's state after the press; false for a button
	// that is not a toggle.
	On bool
}

// Focus gives the button keyboard focus. It returns no Cmd; the result is
// there so a button can stand where any focusable widget does.
func (m *Model) Focus() tui.Cmd {
	m.focused = true
	return nil
}

// Blur takes keyboard focus away, and lets go of a pointer press in
// progress.
func (m *Model) Blur() { m.focused, m.down = false, false }

// Focused reports whether the button has keyboard focus.
func (m Model) Focused() bool { return m.focused }

// Hovered reports whether the pointer is over the button. It is only ever
// true with Mouse on and the Program reporting all pointer motion.
func (m Model) Hovered() bool { return m.hovered }

// Down reports whether a pointer press on the button is in progress: the
// left button went down inside it and has not come up yet.
func (m Model) Down() bool { return m.down }

// On reports whether a toggle button is on. It is always false for a button
// that is not a toggle.
func (m Model) On() bool { return m.Toggle && m.on }

// SetOn turns a toggle button on or off without a press: no PressedMsg is
// delivered. It does nothing to a button that is not a toggle.
func (m *Model) SetOn(on bool) { m.on = on && m.Toggle }

// pressable reports whether the button can be pressed at all.
func (m Model) pressable() bool { return !m.Disabled && !m.Loading }

// press is the result of a press: a toggle flips, and the Cmd delivers
// PressedMsg with the state the press left it in.
func (m Model) press() (Model, tui.Cmd) {
	if m.Toggle {
		m.on = !m.on
	}
	id, on := m.ID, m.On()
	return m, func() tui.Msg { return PressedMsg{ID: id, On: on} }
}

// Update presses the button on Enter or Space when it is focused, and, with
// Mouse on, on a left click inside Bounds. A click is a press and a release
// that both land inside the button: a pointer dragged off it before the
// release presses nothing. A disabled or loading button ignores both.
func (m Model) Update(msg tui.Msg) (Model, tui.Cmd) {
	if ev, ok := msg.(tui.MouseEvent); ok {
		return m.updateMouse(ev)
	}
	if m.focused && m.pressable() && keymap.Matches(msg, m.keys().Press) {
		return m.press()
	}
	return m, nil
}

func (m Model) updateMouse(ev tui.MouseEvent) (Model, tui.Cmd) {
	if !m.Mouse {
		return m, nil
	}
	inside := m.Bounds.Contains(ev.X, ev.Y)
	switch ev.Action {
	case tui.MouseActionMotion:
		m.hovered = inside
	case tui.MouseActionPress:
		if ev.Button == tui.MouseButtonLeft {
			m.hovered = inside
			m.down = inside && m.pressable()
		}
	case tui.MouseActionRelease:
		was := m.down
		m.down = false
		if was && inside && m.pressable() {
			return m.press()
		}
	}
	return m, nil
}

// pad is the number of spaces each side of the label.
func (m Model) pad() int {
	switch m.Size {
	case SizeSmall:
		return 0
	case SizeLarge:
		return 3
	}
	return 1
}

// label is Label with terminal control sequences removed, so text from an
// untrusted source cannot carry an escape into the frame.
func (m Model) label() string { return ansi.Sanitize(m.Label) }

// text is the label as drawn: with an ellipsis while loading.
func (m Model) text() string {
	if m.Loading {
		return m.label() + m.themed().GlyphSet().Ellipsis
	}
	return m.label()
}

// brackets returns the characters either side of the padded label. A
// disabled button is drawn in parentheses, so it reads as unavailable
// without its dimmed colour. A button that has focus is drawn in angle
// brackets. Ghost and link buttons have none at rest; held down without
// focus, they get the square ones.
func (m Model) brackets() (left, right string) {
	switch {
	case m.Disabled:
		return "(", ")"
	case m.focused:
		return "<", ">"
	case (m.Variant == VariantGhost || m.Variant == VariantLink) && !m.down:
		return " ", " "
	}
	return "[", "]"
}

// style is the style of the whole button in its current state.
func (m Model) style() ansi.Style {
	t := m.themed()
	st := t.ResolvedStates()
	var s ansi.Style
	switch m.Variant {
	case VariantSecondary, VariantGhost:
		s = ansi.NewStyle().Foreground(t.Text)
	case VariantDestructive:
		s = ansi.NewStyle().Foreground(t.Error)
	case VariantLink:
		s = ansi.NewStyle().Foreground(t.Primary).Underline()
	default:
		s = ansi.NewStyle().Foreground(t.Primary)
	}
	switch {
	case m.Disabled:
		return st.Disabled
	case m.down:
		return s.Reverse()
	case m.focused:
		return s.Bold()
	case m.hovered:
		return s.Bold()
	}
	return s
}

// leftPad is what stands between the left bracket and the label: the
// padding, and for a toggle button the cell that holds the on marker, which
// takes the place of the last padding space (a small toggle, which has no
// padding, gains the cell).
func (m Model) leftPad() string {
	pad := strings.Repeat(" ", m.pad())
	if !m.Toggle {
		return pad
	}
	mark := " "
	if m.on {
		mark = m.themed().GlyphSet().Dot
	}
	if pad == "" {
		return mark
	}
	return pad[1:] + mark
}

// View renders the button on one row. Its width is the label's width plus
// two brackets and the padding of its Size, in every state, so a button does
// not move its neighbours when it is focused, pressed or toggled; only
// Loading, which adds an ellipsis, makes it one cell wider.
func (m Model) View() string {
	left, right := m.brackets()
	return m.style().Render(left + m.leftPad() + m.text() + strings.Repeat(" ", m.pad()) + right)
}

// Width is the number of cells View takes.
func (m Model) Width() int {
	w := ansi.Width(m.text()) + 2 + 2*m.pad()
	if m.Toggle && m.pad() == 0 {
		w++
	}
	return w
}

// Compile-time proof that Model satisfies tui.Component[Model] — see
// tui.Component's doc comment for what this contract means and why.
var _ tui.Component[Model] = Model{}

// Linearize renders the button as plain text for accessible output (see
// tui.Linearizer): the label, the word "button", and its state, e.g.
// "Save, button, focused" or "Delete, button, unavailable". A toggle button
// reads "Bold, toggle button, on".
func (m Model) Linearize() string {
	out := m.label() + ", button"
	if m.Toggle {
		out = m.label() + ", toggle button, off"
		if m.on {
			out = m.label() + ", toggle button, on"
		}
	}
	switch {
	case m.Disabled:
		out += ", unavailable"
	case m.Loading:
		out += ", busy"
	case m.focused:
		out += ", focused"
	}
	return out
}
