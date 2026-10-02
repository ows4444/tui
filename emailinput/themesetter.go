package emailinput

import (
	"github.com/ows4444/tui"
	"github.com/ows4444/tui/theme"
)

// SetTheme returns m with t applied to the embedded textinput.Model. It is
// defined here so it returns this package's Model; the promoted
// textinput.Model.SetTheme would return the embedded type and drop the
// wrapper's filtering.
func (m Model) SetTheme(t theme.Theme) Model {
	m.Model = m.Model.SetTheme(t)
	return m
}

// Compile-time proof that Model satisfies tui.ThemeSetter[Model].
var _ tui.ThemeSetter[Model] = Model{}
