package menu

import (
	"github.com/ows4444/tui/picker"
	"github.com/ows4444/tui/theme"
)

// Tokens returns the instance's colour overrides as set by WithTokens.
func (m Model) Tokens() theme.Tokens { return m.tokens }

// WithTokens returns m with tok as its per-instance colour override, applied
// to every level, including levels opened later. Nil fields inherit from the
// theme. Theme-wide overrides registered under theme.ComponentMenu are applied
// by SetTheme.
func (m Model) WithTokens(tok theme.Tokens) Model {
	m.tokens = tok
	stack := make([]picker.Model, len(m.stack))
	for i, pk := range m.stack {
		stack[i] = pk.WithTokens(tok)
	}
	m.stack = stack
	return m
}
