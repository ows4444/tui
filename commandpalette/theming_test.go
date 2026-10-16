package commandpalette

import (
	"testing"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
)

// SetTheme applies a theme, WithTokens overrides roles for this instance, and
// Tokens reports what the widget renders with.
func TestThemeAndTokens(t *testing.T) {
	m := New()
	light := theme.LightTheme()
	m = m.SetTheme(light)
	if m.Theme.Primary != light.Primary {
		t.Errorf("SetTheme did not apply the theme")
	}
	red := ansi.Red
	m = m.WithTokens(theme.Tokens{Accent: red})
	if got := m.Tokens().Accent; got != red {
		t.Errorf("Tokens().Accent = %v, want the override %v", got, red)
	}
	// Overriding again replaces the first override.
	m = m.WithTokens(theme.Tokens{})
	if got := m.Tokens().Accent; got == red {
		t.Errorf("a second WithTokens did not replace the first")
	}
}
