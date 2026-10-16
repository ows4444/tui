package wizard

import (
	"testing"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
)

// WithTokens overrides roles for this instance and Tokens reports them under a
// theme.
func TestTokens(t *testing.T) {
	m := New("One", "Two")
	th := theme.DarkTheme()
	red := ansi.Red
	m = m.WithTokens(theme.Tokens{Accent: red})
	if got := m.Tokens(th).Accent; got != red {
		t.Errorf("Tokens().Accent = %v, want %v", got, red)
	}
	m = m.WithTokens(theme.Tokens{})
	if got := m.Tokens(th).Accent; got == red {
		t.Error("a second WithTokens did not replace the first")
	}
}
