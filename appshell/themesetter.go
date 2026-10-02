package appshell

import (
	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
)

// SetTheme returns m with t applied to the shell and forwarded to its
// children: the header reads m.Theme, and the Input's text and placeholder
// styles are rebuilt from t's Text and Muted roles. The Content viewport
// renders the caller's own text and has no theme of its own. It makes Model a
// tui.ThemeSetter, so tui.WithTheme reaches the shell without app code
// forwarding the theme.
func (m Model) SetTheme(t theme.Theme) Model {
	m.Theme = t
	return m.restyleInput()
}

// restyleInput rebuilds the Input's text and placeholder styles from the
// resolved theme.
func (m Model) restyleInput() Model {
	t := m.themed()
	m.Input.TextStyle = ansi.NewStyle().Foreground(t.Text)
	m.Input.PlaceholderStyle = ansi.NewStyle().Foreground(t.Muted)
	return m
}

// Compile-time proof that Model satisfies tui.ThemeSetter[Model].
var _ tui.ThemeSetter[Model] = Model{}
