package textarea

import (
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
)

// Tokens returns the instance's colour overrides as set by WithTokens. Roles
// not overridden are nil: the widget's colours come from SetTheme (which
// honours theme.WithTokens(theme.ComponentTextArea, ...)) or its style fields.
func (m Model) Tokens() theme.Tokens { return m.tokens }

// WithTokens returns m with tok as its per-instance colour override: Text
// recolours the text and Muted the placeholder. Nil fields leave the
// current styles alone, and a later SetTheme keeps the override applied.
func (m Model) WithTokens(tok theme.Tokens) Model {
	m.tokens = tok
	if tok.Text != nil {
		m.TextStyle = ansi.NewStyle().Foreground(tok.Text)
	}
	if tok.Muted != nil {
		m.PlaceholderStyle = ansi.NewStyle().Faint().Foreground(tok.Muted)
	}
	return m
}
