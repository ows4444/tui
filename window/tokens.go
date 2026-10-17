package window

import (
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
)

// borderSet is the characters of a window's border.
type borderSet struct{ Top, Left, Right, Bottom, TopLeft, TopRight, BottomLeft, BottomRight string }

// themeView is what a frame needs from the theme, resolved once per desktop.
type themeView struct {
	border                         borderSet
	edge, muted, focus, focusTitle ansi.Style
	cross                          string
}

// resolved is the theme the widget renders with: m.Theme with the overrides
// registered for theme.ComponentWindow and then the instance tokens applied.
func (m Model) resolved() theme.Theme {
	return m.Theme.Resolve(theme.ComponentWindow, m.tokens)
}

func (m Model) themed() themeView {
	t := m.resolved()
	b := t.Border
	if b == (theme.Border{}) { // a Plain theme draws no border; the window still needs edges
		b = theme.Border{Top: "-", Bottom: "-", Left: "|", Right: "|", TopLeft: "+", TopRight: "+", BottomLeft: "+", BottomRight: "+"}
	}
	return themeView{
		border:     borderSet{b.Top, b.Left, b.Right, b.Bottom, b.TopLeft, b.TopRight, b.BottomLeft, b.BottomRight},
		edge:       ansi.NewStyle().Foreground(t.BorderColor),
		muted:      ansi.NewStyle().Foreground(t.Muted),
		focus:      ansi.NewStyle().Foreground(t.Primary),
		focusTitle: ansi.NewStyle().Foreground(t.Primary).Bold(),
		cross:      t.Glyphs.Resolved().Cross,
	}
}

// Tokens returns the colour tokens the widget renders with: its theme's roles,
// overridden by any theme.WithTokens(theme.ComponentWindow, ...) and then by
// WithTokens.
func (m Model) Tokens() theme.Tokens { return m.resolved().TokensFor("") }

// WithTokens returns m with tok as its per-instance colour override. Nil
// fields inherit from the theme, so only the roles tok names change; the
// theme and every other widget are untouched. A second call replaces the
// first.
func (m Model) WithTokens(tok theme.Tokens) Model {
	m.tokens = tok
	return m
}
