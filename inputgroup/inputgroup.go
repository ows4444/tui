// Package inputgroup is a text field with fixed text before and after it: a
// unit, a currency sign, the "https://" and ".com" either side of a domain.
// The field is a textinput.Model and types, moves and pastes exactly as one;
// the group draws the two additions, keeps the one after the field in a
// steady column, and reports the cursor's cell past the one before it.
//
// Stability: experimental. Its API may change in any minor release.
package inputgroup

import (
	"strings"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/textinput"
	"github.com/ows4444/tui/theme"
)

// Model wraps a textinput.Model. Its promoted fields are the field's own:
// Width is the field's width between the two additions, and Mouse and Bounds
// describe the field alone, so Bounds starts after Prefix.
type Model struct {
	textinput.Model

	// Prefix is drawn before the field, in the muted colour. Put any space
	// that should stand between it and the field in the string: "$ ".
	Prefix string
	// Suffix is drawn after the field, in the muted colour.
	Suffix string
	// Theme supplies the colour of Prefix and Suffix. SetTheme sets it and
	// the field's own styles together.
	Theme theme.Theme
}

// New returns a group around an empty field, with prefix before it and
// suffix after it.
func New(prefix, suffix string) Model {
	return Model{Model: textinput.New(), Prefix: prefix, Suffix: suffix, Theme: theme.DarkTheme()}
}

// Update passes msg to the field. The additions take no input.
func (m Model) Update(msg tui.Msg) (Model, tui.Cmd) {
	next, cmd := m.Model.Update(msg)
	m.Model = next
	return m, cmd
}

func (m Model) prefix() string { return ansi.Sanitize(m.Prefix) }
func (m Model) suffix() string { return ansi.Sanitize(m.Suffix) }

// FullValue returns Prefix, the field's value and Suffix as one string:
// "https://" + "example" + ".com".
func (m Model) FullValue() string { return m.Prefix + m.Value() + m.Suffix }

// addon is the style of the two additions.
func (m Model) addon() ansi.Style {
	return ansi.NewStyle().Foreground(m.Theme.Resolve(theme.ComponentTextInput, m.Model.Tokens()).Muted)
}

// fieldWidth is the number of cells the field takes in View: Width and the
// cell kept for the cursor after the text, or the field's own length when
// Width is 0.
func (m Model) fieldWidth() int {
	if m.Width > 0 {
		return ansi.Width(m.Prompt) + m.Width + 1
	}
	return ansi.Width(m.Model.View())
}

// View renders Prefix, the field and Suffix on one row. With a Width set the
// field is padded to it, so Suffix stays in one column while the text is
// typed.
func (m Model) View() string {
	field := m.Model.View()
	pad := max(m.fieldWidth()-ansi.Width(field), 0)
	out := field + strings.Repeat(" ", pad)
	if p := m.prefix(); p != "" {
		out = m.addon().Render(p) + out
	}
	if s := m.suffix(); s != "" {
		out += m.addon().Render(s)
	}
	return out
}

// CursorCell returns the cell the cursor occupies in View: the field's own,
// moved right by the width of Prefix. ok is false while the field is not
// focused.
func (m Model) CursorCell() (x, y int, ok bool) {
	x, y, ok = m.Model.CursorCell()
	return x + ansi.Width(m.prefix()), y, ok
}

// Compile-time proofs that Model satisfies the contracts a field does.
var (
	_ tui.Component[Model] = Model{}
	_ tui.CursorProvider   = Model{}
)

// Linearize renders the group as plain text for accessible output (see
// tui.Linearizer): the field's own line, then what stands before and after
// it, e.g. `Text field, value "example", before it "https://", after it
// ".com"`.
func (m Model) Linearize() string {
	out := m.Model.Linearize()
	if p := strings.TrimSpace(m.prefix()); p != "" {
		out += `, before it "` + p + `"`
	}
	if s := strings.TrimSpace(m.suffix()); s != "" {
		out += `, after it "` + s + `"`
	}
	return out
}
