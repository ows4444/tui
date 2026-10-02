package textarea

import (
	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
)

// SetTheme returns m with its text, placeholder and cursor styles taken from
// t: the text in t.Text, the placeholder faint in t.Muted, and the cursor in
// reverse video, which needs no colour to stay visible. It makes Model a
// tui.ThemeSetter, so a root model can forward the Program's theme (see
// tui.WithTheme). The style fields stay exported; set them after SetTheme to
// override.
func (m Model) SetTheme(t theme.Theme) Model {
	t = t.Resolve(theme.ComponentTextArea, m.tokens)
	m.TextStyle = ansi.NewStyle().Foreground(t.Text)
	m.PlaceholderStyle = ansi.NewStyle().Faint().Foreground(t.Muted)
	m.CursorStyle = ansi.NewStyle().Reverse()
	return m
}

// Compile-time proof that Model satisfies tui.ThemeSetter[Model].
var _ tui.ThemeSetter[Model] = Model{}
