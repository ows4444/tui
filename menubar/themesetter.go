package menubar

import "github.com/ows4444/tui/theme"

// SetTheme returns m with t applied to the bar and its open dropdown. It makes
// Model a tui.ThemeSetter, so a root model can forward the Program's theme
// (see tui.WithTheme).
func (m Model) SetTheme(t theme.Theme) Model {
	m.Theme = t
	m.drop = m.drop.SetTheme(t)
	return m
}
