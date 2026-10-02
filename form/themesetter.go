package form

import (
	"github.com/ows4444/tui"
	"github.com/ows4444/tui/theme"
)

// SetTheme returns m with t applied. It makes Model a tui.ThemeSetter, so a
// root model can forward the Program's theme (see tui.WithTheme).
func (m Model) SetTheme(t theme.Theme) Model {
	m.Theme = t
	// A theme-level override for the form also reaches its inputs.
	if r := t.Resolve(theme.ComponentForm, theme.Tokens{}); r != t {
		m = m.clone()
		for i := range m.inputs {
			m.inputs[i].text = m.inputs[i].text.SetTheme(r)
			m.inputs[i].secret = m.inputs[i].secret.SetTheme(r)
		}
	}
	return m
}

// Compile-time proof that Model satisfies tui.ThemeSetter[Model].
var _ tui.ThemeSetter[Model] = Model{}
