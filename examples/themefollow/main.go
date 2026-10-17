// Command themefollow shows a fixed theme that follows the terminal between
// light and dark. theme.Pair gives Dracula on a dark background and its
// derived light twin on a light one; WithColorSchemeReports makes a terminal
// that supports DECSET 2031 say when the user switches.
package main

import (
	"fmt"
	"os"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
)

type model struct {
	theme theme.Theme
	width int
}

func newModel() model { return model{theme: theme.DraculaTheme()} }

func (m model) Init() tui.Cmd { return nil }

// SetTheme is called by the Program with the Pair's Dark or Light theme: once
// before the first frame and again whenever the terminal switches.
func (m model) SetTheme(t theme.Theme) tui.Model {
	m.theme = t
	return m
}

func (m model) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	if r, ok := msg.(tui.ResizeMsg); ok {
		m.width = r.Width
	}
	if k, ok := msg.(tui.Key); ok {
		if k.Type == tui.KeyCtrlC || (k.Type == tui.KeyRunes && k.Text == "q") {
			return m, tui.Quit()
		}
	}
	return m, nil
}

func (m model) View() string {
	title := ansi.NewStyle().Bold().Foreground(m.theme.Primary)
	ok := ansi.NewStyle().Foreground(m.theme.Success)
	warn := ansi.NewStyle().Foreground(m.theme.Warning)
	muted := ansi.NewStyle().Foreground(m.theme.Muted)
	help := "Switch the terminal between light and dark; q quits."
	if m.width > 0 {
		help = ansi.Wrap(help, m.width)
	}
	return fmt.Sprintf("%s\n\n  %s  %s\n\n%s",
		title.Render("Theme follow"),
		ok.Render("saved"), warn.Render("unsaved changes"),
		muted.Render(help))
}

func main() {
	p := tui.NewProgram(newModel(),
		tui.WithTheme(theme.Pair(theme.DraculaTheme())),
		tui.WithColorSchemeReports(true))
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
