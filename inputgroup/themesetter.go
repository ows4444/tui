package inputgroup

import (
	"github.com/ows4444/tui"
	"github.com/ows4444/tui/theme"
)

// SetTheme returns m with t applied to the field and to the two additions.
// It is defined here so it returns this package's Model; the promoted
// textinput.Model.SetTheme would return the embedded type and drop the
// additions.
func (m Model) SetTheme(t theme.Theme) Model {
	m.Model = m.Model.SetTheme(t)
	m.Theme = t
	return m
}

// Compile-time proof that Model satisfies tui.ThemeSetter[Model].
var _ tui.ThemeSetter[Model] = Model{}

// Tokens returns the field's colour overrides, as set by WithTokens. The
// group shares theme.ComponentTextInput with textinput; the additions are
// drawn in its muted colour.
func (m Model) Tokens() theme.Tokens { return m.Model.Tokens() }

// WithTokens returns m with tok as its per-instance colour override, applied
// to the field and to the additions. It is defined here so it returns this
// package's Model.
func (m Model) WithTokens(tok theme.Tokens) Model {
	m.Model = m.Model.WithTokens(tok)
	return m
}
