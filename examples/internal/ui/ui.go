// Package ui is what the example programs share: the Program they all start
// from, the theme the Program hands to a model, and the plain text a screen
// reader gets. Each example is otherwise a package main of its own that you can
// read and copy on its own; this is the part that is the same in all of them.
package ui

import (
	"strings"
	"unicode"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/theme"
)

// NewProgram is tui.NewProgram with what every example wants first: the Program
// owns the theme and swaps between the dark and the light theme to match the
// terminal (WithTheme, WithColorSchemeReports), and ACCESSIBLE=1 or TERM=dumb
// switches to screen-reader output (WithAccessibleAuto). A model that has no
// SetTheme method simply ignores the theme. Later options win.
func NewProgram(m tui.Model, opts ...tui.ProgramOption) *tui.Program {
	base := []tui.ProgramOption{tui.WithTheme(theme.DefaultAuto()), tui.WithColorSchemeReports(true), tui.WithAccessibleAuto()}
	return tui.NewProgram(m, append(base, opts...)...)
}

// Theming holds the theme a Program gave a model, for the model to keep as a
// field. Its zero value is the dark theme, which is what a model sees before the
// Program has chosen one and in tests.
type Theming struct {
	t   theme.Theme
	set bool
}

// Set returns s holding t.
func (s Theming) Set(t theme.Theme) Theming { return Theming{t: t, set: true} }

// Get returns the theme the Program chose, or the dark theme if it has not.
func (s Theming) Get() theme.Theme {
	if !s.set {
		return theme.DarkTheme()
	}
	return s.t
}

// Linearize turns a drawn screen into the plain text a screen reader should
// get: no colours, no padding, no box drawing and no blank or purely decorative
// lines. It is a stand-in for a Linearize method that knows its own widgets, and
// good enough for a screen that is mostly text.
func Linearize(view string) string {
	var out []string
	for _, line := range strings.Split(ansi.StripANSI(view), "\n") {
		line = strings.TrimFunc(line, func(r rune) bool { return unicode.IsSpace(r) || decoration(r) })
		if line == "" {
			continue
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

// decoration reports whether r only draws: box and block characters, rules and
// the dots of a background pattern.
func decoration(r rune) bool {
	switch {
	case r >= 0x2500 && r <= 0x259f, r == '·', r == '|':
		return true
	}
	return false
}

// Help is one line of a help screen: a key (or a gesture) and what it does.
type Help struct{ Key, Does string }

// HelpScreen lays out a help screen: a title, one line per key, and how to get
// back. A program shows it in place of its own screen while help is open.
func HelpScreen(th theme.Theme, title string, lines ...Help) layout.Node {
	width := 0
	for _, l := range lines {
		width = max(width, ansi.Width(l.Key))
	}
	kids := []layout.FlexChild{{Node: layout.Block(" " + ansi.NewStyle().Bold().Foreground(th.Primary).Render(title))}, {Node: layout.Block("")}}
	for _, l := range lines {
		key := ansi.NewStyle().Bold().Foreground(th.Primary).Render(l.Key)
		kids = append(kids, layout.FlexChild{Node: layout.Block(" " + key + strings.Repeat(" ", width-ansi.Width(l.Key)+2) + l.Does)})
	}
	kids = append(kids, layout.FlexChild{Node: layout.Block("")}, layout.FlexChild{Node: layout.Block(" " + ansi.NewStyle().Faint().Render("Press any key to close this help."))})
	return layout.Column(0, kids...)
}
