// Package maskedinput is a thin wrapper over textinput.Model that masks
// every rendered character of a non-empty value with a configurable
// rune. Update's behavior (typing, deletion, navigation, paste, focus) is
// reused unchanged from textinput.Model — Model.Update exists only to
// re-wrap the result so Mask survives the round trip (see its doc
// comment); only View differs otherwise.
package maskedinput

import (
	"github.com/ows4444/tui/internal/a11y"
	"strconv"
	"strings"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/textinput"
	"github.com/ows4444/tui/theme"
)

// defaultMask is the rune Mask is set to by New.
const defaultMask = '•'

// Model wraps a textinput.Model, masking its rendered value with Mask.
type Model struct {
	textinput.Model

	// Mask is the rune each character of a non-empty value is rendered
	// as. Defaults to '•' (set by New); callers may change it at any
	// time. While it is left at that default, the mask follows Theme (an
	// ASCII theme draws '*'); any other rune is drawn as given.
	Mask rune

	// Theme supplies the default mask glyph (theme.Glyphs.Mask). New sets
	// theme.DarkTheme(); a zero Theme draws the default '•'.
	Theme theme.Theme
}

// New returns a Model with textinput's default styling and Mask set to
// the default '•'.
func New() Model {
	return Model{Model: textinput.New(), Mask: defaultMask, Theme: theme.DarkTheme()}
}

// Update forwards msg to the embedded textinput.Model unchanged and
// re-wraps the result, preserving Mask — without this override, the
// promoted textinput.Model.Update would be selected instead, whose return
// type is textinput.Model, not Model: the standard `m, cmd := m.Update(msg)`
// pattern every other widget in this library supports would silently
// discard Mask on the first call.
func (m Model) Update(msg tui.Msg) (Model, tui.Cmd) {
	next, cmd := m.Model.Update(msg)
	m.Model = next
	return m, cmd
}

// View renders every character of a non-empty value masked with Mask
// (criterion #516), while Value() still returns the real, unmasked typed
// text. An empty value still renders Placeholder unmasked, matching
// textinput.Model's empty-value behavior.
func (m Model) View() string {
	if len(m.Value()) == 0 {
		return m.Model.View()
	}

	mask := m.Mask
	if mask == 0 {
		mask = defaultMask
	}

	// Render through a copy whose value has been replaced with masked
	// runes of the same length, so textinput's own cursor/width/scroll
	// logic (styling, windowing) is reused unchanged.
	masked := m.Model
	glyph := string(mask)
	if mask == defaultMask {
		glyph = m.Theme.GlyphSet().Mask
	}
	masked.SetValue(strings.Repeat(glyph, len([]rune(m.Value()))))
	masked.SetCursor(m.Cursor())
	return masked.View()
}

// Compile-time proof that Model satisfies tui.Component[Model] — see
// tui.Component's doc comment for what this contract means and why.
var _ tui.Component[Model] = Model{}

// Linearize renders the field as one plain-text line for accessible output
// (see tui.Linearizer): its label, "masked field", ", focused" when it has focus, and
// how many characters were entered, e.g. `Password, masked field, 8 characters
// entered`. It never includes the typed value. This override is required:
// Model embeds textinput.Model, whose own Linearize would otherwise be
// promoted and speak the real value.
func (m Model) Linearize() string {
	head := "Masked field"
	if l := a11y.PromptLabel(m.Prompt); l != "" {
		head = l + ", masked field"
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
