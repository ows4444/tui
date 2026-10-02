package commandpalette

import (
	"github.com/ows4444/tui"
	"github.com/ows4444/tui/theme"
)

// SetTheme returns m with t applied. It makes Model a tui.ThemeSetter, so a
// root model can forward the Program's theme (see tui.WithTheme).
func (m Model) SetTheme(t theme.Theme) Model {
	m.Theme = t
	// A theme-level override for this component also reaches the embedded input.
	if r := t.Resolve(theme.ComponentCommandPalette, theme.Tokens{}); r != t {
		m.Input = m.Input.SetTheme(r)
	}
	return m
}

// Compile-time proof that Model satisfies tui.ThemeSetter[Model].
var _ tui.ThemeSetter[Model] = Model{}
