// Package toggle is an on/off switch with a label, the Switch of web
// component libraries ("switch" is a Go keyword and cannot name a package).
// Space or Enter flips it when it has focus, Left turns it off and Right
// turns it on, and with the mouse on so does a click.
//
// Stability: experimental. Its API may change in any minor release.
package toggle

import (
	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/hittest"
	"github.com/ows4444/tui/keymap"
	"github.com/ows4444/tui/theme"
)

// Model is one switch. The zero value is not ready to use; build one with
// New.
type Model struct {
	// ID is carried by ChangedMsg, to tell one switch's change from
	// another's.
	ID string
	// Label is the text after the switch. An empty Label draws the switch
	// alone.
	Label string
	// Disabled stops the switch being changed; it is drawn faint.
	Disabled bool
	Theme    theme.Theme

	// tokens is the per-instance colour override set by WithTokens.
	tokens theme.Tokens
	// KeyMap holds the keys for each action. New fills it with DefaultKeyMap;
	// a Model built as a struct literal with a zero KeyMap behaves as if it
	// held DefaultKeyMap.
	KeyMap KeyMap

	// Mouse, when true, makes Update handle tui.MouseEvent: a left press on
	// the switch or its label flips it. Off (the default) ignores the mouse.
	Mouse bool
	// Bounds is the screen rectangle where the app draws the switch.
	Bounds hittest.Rect

	on      bool
	focused bool
}

// New returns a switch labelled label, turned off.
func New(label string) Model {
	return Model{Label: label, Theme: theme.DarkTheme(), KeyMap: DefaultKeyMap()}
}

// KeyMap names the keys of each action of a Model.
type KeyMap struct {
	Flip keymap.Binding // turn the switch on if it is off, and off if it is on
	Off  keymap.Binding // turn the switch off
	On   keymap.Binding // turn the switch on
}

// DefaultKeyMap returns Space or Enter, Left and Right.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Flip: keymap.NewBinding("switch", "space", "enter"),
		Off:  keymap.NewBinding("off", "left"),
		On:   keymap.NewBinding("on", "right"),
	}
}

func (m Model) keys() KeyMap {
	km := m.KeyMap
	if len(km.Flip.Keys)+len(km.Off.Keys)+len(km.On.Keys) == 0 {
		return DefaultKeyMap()
	}
	return km
}

// Bindings returns the actions the switch honours, with descriptions, for
// help text. A disabled one has none.
func (m Model) Bindings() []keymap.Binding {
	if m.Disabled {
		return nil
	}
	km := m.keys()
	return []keymap.Binding{km.Flip, km.Off, km.On}
}

// ChangedMsg is delivered (via the Cmd Update returns) when a key or a click
// turns the switch on or off.
type ChangedMsg struct {
	// ID is the switch's ID.
	ID string
	// On is the switch's state after the change.
	On bool
}

// On reports whether the switch is on.
func (m Model) On() bool { return m.on }

// SetOn turns the switch on or off without a key press: no ChangedMsg is
// delivered.
func (m *Model) SetOn(on bool) { m.on = on }

// Focus gives the switch keyboard focus. It returns no Cmd; the result is
// there so a switch can stand where any focusable widget does.
func (m *Model) Focus() tui.Cmd {
	m.focused = true
	return nil
}

// Blur takes keyboard focus away.
func (m *Model) Blur() { m.focused = false }

// Focused reports whether the switch has keyboard focus.
func (m Model) Focused() bool { return m.focused }

// set turns the switch to on and returns the Cmd that reports it, or nil
// when it was already there.
func (m Model) set(on bool) (Model, tui.Cmd) {
	if m.on == on {
		return m, nil
	}
	m.on = on
	msg := ChangedMsg{ID: m.ID, On: on}
	return m, func() tui.Msg { return msg }
}

// Update flips the switch on Space or Enter, turns it off on Left and on on
// Right, when it has focus, and, with Mouse on, flips it on a left press
// inside Bounds. A disabled switch ignores both.
func (m Model) Update(msg tui.Msg) (Model, tui.Cmd) {
	if m.Disabled {
		return m, nil
	}
	if ev, ok := msg.(tui.MouseEvent); ok {
		if m.Mouse && ev.Action == tui.MouseActionPress && ev.Button == tui.MouseButtonLeft &&
			m.Bounds.Contains(ev.X, ev.Y) {
			return m.set(!m.on)
		}
		return m, nil
	}
	if !m.focused {
		return m, nil
	}
	km := m.keys()
	switch {
	case keymap.Matches(msg, km.Flip):
		return m.set(!m.on)
	case keymap.Matches(msg, km.Off):
		return m.set(false)
	case keymap.Matches(msg, km.On):
		return m.set(true)
	}
	return m, nil
}

func (m Model) label() string { return ansi.Sanitize(m.Label) }

// View renders a track with a knob at its left end when off, "(○  )", and at
// its right end, filled, when on, "(  ●)", followed by the label, so the
// state reads without colour in two ways. A switch that has focus is drawn
// with angle brackets, "<  ●>", in the focus style; a disabled one is faint.
func (m Model) View() string {
	t := m.themed()
	st := t.ResolvedStates()
	g := t.GlyphSet()
	track := g.DotEmpty + "  "
	body := ansi.NewStyle().Foreground(t.Muted)
	if m.on {
		track = "  " + g.Dot
		body = ansi.NewStyle().Foreground(t.Success)
	}
	open, shut := "(", ")"
	text := ansi.NewStyle().Foreground(t.Text)
	switch {
	case m.Disabled:
		body, text = st.Disabled.Faint(), st.Disabled.Faint()
	case m.focused:
		open, shut = "<", ">"
		body = body.Bold()
	}
	edge := body
	if m.focused && !m.Disabled {
		edge = st.Focus.Bold()
	}
	out := edge.Render(open) + body.Render(track) + edge.Render(shut)
	if l := m.label(); l != "" {
		out += " " + text.Render(l)
	}
	return out
}

// Width is the number of cells View takes.
func (m Model) Width() int {
	if l := m.label(); l != "" {
		return 6 + ansi.Width(l)
	}
	return 5
}

// Compile-time proof that Model satisfies tui.Component[Model] — see
// tui.Component's doc comment for what this contract means and why.
var _ tui.Component[Model] = Model{}

// Linearize renders the switch as plain text for accessible output (see
// tui.Linearizer): "Overwrite, switch, on", with ", unavailable" when
// disabled.
func (m Model) Linearize() string {
	out := "switch, off"
	if m.on {
		out = "switch, on"
	}
	if l := m.label(); l != "" {
		out = l + ", " + out
	}
	if m.Disabled {
		out += ", unavailable"
	}
	return out
}
