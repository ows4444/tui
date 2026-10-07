// Package helpscreen is a full-screen key-binding help overlay: a
// bordered box listing widgets.Hint entries one per line via
// widgets.KeyHint, unlike widgets.KeyHints which joins them into a single
// line. Its open/dismiss lifecycle mirrors dialog.Model exactly, the same
// contract popover.Model already reuses.
package helpscreen

import (
	"strings"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/hittest"
	"github.com/ows4444/tui/internal/boxdraw"
	"github.com/ows4444/tui/keymap"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/theme"
	"github.com/ows4444/tui/widgets"
)

// Model is a full-width help overlay. Render composites it over a base
// view when open; when closed, Render returns base unchanged, so a parent
// can unconditionally call it every frame without checking Open itself.
type Model struct {
	// Hints is the list of key bindings shown, one per line, via
	// widgets.KeyHint.
	Hints []widgets.Hint
	Theme theme.Theme

	// tokens is the per-instance colour override set by WithTokens.
	tokens theme.Tokens
	// Mouse, when true, makes Update handle tui.MouseEvent: a left click
	// outside Bounds dismisses the screen, as Esc does; other mouse events are
	// swallowed while it is open. Off (the default) ignores the mouse. There is
	// no wheel handling: the screen does not scroll.
	Mouse bool
	// Bounds is the screen rectangle the help box occupies (top-left at the
	// origin, as many columns as the base view, Hints plus 4 rows). A click
	// inside it does nothing; with a zero Bounds every click dismisses.
	Bounds hittest.Rect

	open bool
}

// New returns an already-open Model listing hints — the common case is
// showing help immediately in response to some trigger, not constructing
// one ahead of time and opening it later (though Show/Hide support that
// too).
func New(hints ...widgets.Hint) Model {
	return Model{Hints: hints, Theme: theme.DarkTheme(), open: true}
}

// FromRegistry returns an open Model listing the bindings r holds in scope
// ("" is the default scope widgets register in), one hint per binding with its
// keys joined by "/". Pass tui.Program.Keymap() so the screen shows the keys
// the focused widget actually honours, including any the app rebound through
// the widget's KeyMap; build it again when the screen is opened (not once at
// start-up) and it follows the focus and the rebinding with no other code.
func FromRegistry(r *keymap.Registry, scope string) Model {
	var hints []widgets.Hint
	if r != nil {
		for _, h := range r.Hints(scope) {
			hints = append(hints, widgets.Hint{Key: h.Key, Action: h.Desc})
		}
	}
	return New(hints...)
}

// Open reports whether the help screen is currently shown.
func (m Model) Open() bool { return m.open }

// Show opens the help screen.
func (m *Model) Show() { m.open = true }

// Hide closes the help screen without dismissing it via Update (no
// DismissedMsg is sent).
func (m *Model) Hide() { m.open = false }

// DismissedMsg is delivered (via the Cmd Update returns) when the help
// screen is dismissed with Enter or Esc.
type DismissedMsg struct{}

// Update dismisses the help screen on Enter or Esc while open; it's a
// no-op (including for Enter/Esc) once closed, so a parent doesn't need
// to guard forwarding to it on Model.Open(). With Mouse on, a left click
// outside Bounds dismisses it too.
func (m Model) Update(msg tui.Msg) (Model, tui.Cmd) {
	if !m.open {
		return m, nil
	}
	if ev, ok := msg.(tui.MouseEvent); ok {
		if m.Mouse && ev.Action == tui.MouseActionPress && ev.Button == tui.MouseButtonLeft &&
			!m.Bounds.Contains(ev.X, ev.Y) {
			m.open = false
			return m, func() tui.Msg { return DismissedMsg{} }
		}
		return m, nil
	}
	key, ok := msg.(tui.Key)
	if !ok {
		return m, nil
	}
	switch key.Type {
	case tui.KeyEnter, tui.KeyEsc:
		m.open = false
		return m, func() tui.Msg { return DismissedMsg{} }
	}
	return m, nil
}

// Render composites the help screen, spanning the full width of base, at
// its top-left corner. If the help screen is closed, it returns base
// unchanged.
func (m Model) Render(base string) string {
	if !m.open {
		return base
	}

	baseLines := strings.Split(base, "\n")
	baseWidth := 0
	for _, l := range baseLines {
		if w := ansi.Width(l); w > baseWidth {
			baseWidth = w
		}
	}

	lines := make([]string, len(m.Hints))
	for i, h := range m.Hints {
		lines[i] = m.hint(h)
	}
	content := strings.Join(lines, "\n")

	// A bordered box with 1 cell of padding on each side adds 4 to its
	// outer width beyond its content width (1 border + 1 padding per
	// side); size the content so the box's outer width matches base's.
	contentWidth := baseWidth - 4
	if contentWidth < 0 {
		contentWidth = 0
	}

	box := boxdraw.Draw(layout.NewBox().BorderColor(m.themed().BorderColor), m.themed().Border, 1, contentWidth, content)

	return layout.Overlay(base, box, 0, 0)
}

// Compile-time proof that Model satisfies tui.Overlay[Model] — see
// tui.Overlay's doc comment for what this contract means and why.
var _ tui.Overlay[Model] = Model{}

// FromKeymap returns an already-open Model listing the bindings of scope in
// reg, so the help screen reads the same registry as the key-hint widget.
func FromKeymap(reg *keymap.Registry, scope string) Model {
	return New(widgets.HintsFromKeymap(reg, scope)...)
}
