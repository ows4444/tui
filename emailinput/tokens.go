package emailinput

import "github.com/ows4444/tui/theme"

// Tokens returns the embedded textinput's colour overrides, as set by
// WithTokens. The widget shares theme.ComponentTextInput with textinput.
func (m Model) Tokens() theme.Tokens { return m.Model.Tokens() }

// WithTokens returns m with tok as its per-instance colour override, applied
// to the embedded textinput.Model. It is defined here so it returns this
// package's Model.
func (m Model) WithTokens(tok theme.Tokens) Model {
	m.Model = m.Model.WithTokens(tok)
	return m
}
