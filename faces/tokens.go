package faces

import (
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
)

// themed is the theme the widget renders with: m.Theme with the overrides
// registered for theme.ComponentFaces and then the instance tokens applied.
func (m Model) themed() theme.Theme {
	return m.Theme.Resolve(theme.ComponentFaces, m.tokens)
}

// colour is the colour the face is drawn in: the Accent given to WithTokens
// if there is one, else Color, else the theme's primary colour.
func (m Model) colour() ansi.Color {
	if m.Color != nil && m.tokens.Accent == nil {
		return m.Color
	}
	return m.themed().Primary
}

// Tokens returns the colour tokens the widget renders with: its theme's roles,
// overridden by any theme.WithTokens(theme.ComponentFaces, ...), then by Color
// for the Accent, and then by WithTokens.
func (m Model) Tokens() theme.Tokens {
	tok := m.Theme.Resolve(theme.ComponentFaces, m.tokens).TokensFor("")
	tok.Accent = m.colour()
	return tok
}

// WithTokens returns m with tok as its per-instance colour override. Nil
// fields inherit from the theme, so only the roles tok names change; the
// theme and every other widget are untouched. A second call replaces the
// first.
func (m Model) WithTokens(tok theme.Tokens) Model {
	m.tokens = tok
	return m
}
