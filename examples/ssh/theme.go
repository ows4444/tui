package main

import (
	"github.com/ows4444/tui"
	"github.com/ows4444/tui/examples/internal/ui"
	"github.com/ows4444/tui/theme"
)

// SetTheme implements tui.Themeable: the session's Program asks the client's
// terminal for its colours and hands over the dark or the light theme. It works
// on a copy, like Update, and the copy shares the hub and the id.
func (l *lobby) SetTheme(t theme.Theme) tui.Model {
	c := *l
	c.th = c.th.Set(t)
	return &c
}

// theme is the theme to draw with.
func (l *lobby) theme() theme.Theme { return l.th.Get() }

// Linearize implements tui.Linearizer: the room as plain lines.
func (l *lobby) Linearize() string { return ui.Linearize(l.View()) }
