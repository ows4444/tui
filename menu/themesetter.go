package menu

import (
	"github.com/ows4444/tui"
	"github.com/ows4444/tui/picker"
	"github.com/ows4444/tui/theme"
)

// SetTheme returns m with t applied to every level, including levels opened
// later. It makes Model a tui.ThemeSetter, so a root model can forward the
// Program's theme (see tui.WithTheme).
func (m Model) SetTheme(t theme.Theme) Model {
	t = t.Resolve(theme.ComponentMenu, theme.Tokens{})
	m.styling, m.hasStyling = t, true
	stack := make([]picker.Model, len(m.stack))
	for i, pk := range m.stack {
		pk.Theme = t
		stack[i] = pk
	}
	m.stack = stack
	return m
}

// Compile-time proof that Model satisfies tui.ThemeSetter[Model].
var _ tui.ThemeSetter[Model] = Model{}
