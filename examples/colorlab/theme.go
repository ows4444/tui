package main

import (
	"github.com/ows4444/tui"
	"github.com/ows4444/tui/examples/internal/ui"
	"github.com/ows4444/tui/theme"
)

// SetTheme implements tui.Themeable: the Program hands over the dark or the
// light theme to match the terminal, at start and whenever the terminal's colour
// scheme changes.
func (m model) SetTheme(t theme.Theme) tui.Model {
	m.th = m.th.Set(t)
	return m
}

// theme is the theme to draw with.
func (m model) theme() theme.Theme { return m.th.Get() }

// Linearize implements tui.Linearizer: the text a screen reader gets under
// WithAccessible (ACCESSIBLE=1), the screen without colours, padding or boxes.
func (m model) Linearize() string { return ui.Linearize(m.View()) }
