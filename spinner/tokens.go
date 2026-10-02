package spinner

import "github.com/ows4444/tui/theme"

// themed is the theme the widget renders with: m.Theme with the overrides
// registered for theme.ComponentSpinner and then the instance tokens applied.
func (m Model) themed() theme.Theme {
	return m.Theme.Resolve(theme.ComponentSpinner, m.tokens)
}

// Tokens returns the colour tokens the widget renders with: its theme's roles,
// overridden by any theme.WithTokens(theme.ComponentSpinner, ...) and then by
// WithTokens.
func (m Model) Tokens() theme.Tokens {
	return m.Theme.Resolve(theme.ComponentSpinner, m.tokens).TokensFor("")
}

// WithTokens returns m with tok as its per-instance colour override. Nil
// fields inherit from the theme, so only the roles tok names change; the
// theme and every other widget are untouched. A second call replaces the
// first.
func (m Model) WithTokens(tok theme.Tokens) Model {
	m.tokens = tok
	return m
}
