// Package passwordinput is a thin wrapper over textinput.Model that masks
// every rendered character of a non-empty value. Update's behavior
// (typing, deletion, navigation, paste, focus) is reused unchanged from
// textinput.Model — Model.Update exists only to re-wrap the result as
// Model instead of textinput.Model (see its doc comment); only View
// differs otherwise.
package passwordinput

import (
	"github.com/ows4444/tui/internal/a11y"
	"strconv"
	"strings"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/textinput"
	"github.com/ows4444/tui/theme"
)

// Mask is the rune each character of a non-empty value is rendered as.
const Mask = '•'

// Model wraps a textinput.Model, masking its rendered value.
type Model struct {
	textinput.Model

	// Theme supplies the mask glyph (theme.Glyphs.Mask): Mask under the
	// default theme, '*' under an ASCII one. New sets theme.DarkTheme(); a zero Theme
	// draws Mask.
	Theme theme.Theme
}

// New returns a Model with textinput's default styling.
func New() Model {
	in := textinput.New()
	in.DisableCopy = true // a secret never goes to the clipboard
	return Model{Model: in, Theme: theme.DarkTheme()}
}

// Update forwards msg to the embedded textinput.Model unchanged and
// re-wraps the result — without this override, the promoted
// textinput.Model.Update would be selected instead, whose return type is
// textinput.Model, not Model: the standard `m, cmd := m.Update(msg)`
// pattern every other widget in this library supports would silently
// change m's type out from under a caller expecting Model.
func (m Model) Update(msg tui.Msg) (Model, tui.Cmd) {
	next, cmd := m.Model.Update(msg)
	m.Model = next
	return m, cmd
}

// View renders every character of a non-empty value masked with Mask
// (criterion #337), while Value() still returns the real, unmasked typed
// text. An empty value still renders Placeholder unmasked, matching
// textinput.Model's empty-value behavior (criterion #338).
func (m Model) View() string {
	if len(m.Value()) == 0 {
		return m.Model.View()
	}

	// Render through a copy whose value has been replaced with masked
	// runes of the same length, so textinput's own cursor/width/scroll
	// logic (styling, windowing) is reused unchanged.
	masked := m.Model
	masked.SetValue(strings.Repeat(m.Theme.GlyphSet().Mask, len([]rune(m.Value()))))
	masked.SetCursor(m.Cursor())
	return masked.View()
}

// Compile-time proof that Model satisfies tui.Component[Model] — see
// tui.Component's doc comment for what this contract means and why.
var _ tui.Component[Model] = Model{}

// Linearize renders the field as one plain-text line for accessible output
// (see tui.Linearizer): its label, "password field", ", focused" when it has focus, and
// how many characters were entered, e.g. `Password, password field, 8 characters
// entered`. It never includes the typed value. This override is required:
// Model embeds textinput.Model, whose own Linearize would otherwise be
// promoted and speak the real value.
func (m Model) Linearize() string {
	head := "Password field"
	if l := a11y.PromptLabel(m.Prompt); l != "" {
		head = l + ", password field"
	}
	if m.Focused() {
		head += ", focused"
	}
	switch n := len([]rune(m.Value())); n {
	case 0:
		return head + ", empty"
	case 1:
		return head + ", 1 character entered"
	default:
		return head + ", " + strconv.Itoa(n) + " characters entered"
	}
}
