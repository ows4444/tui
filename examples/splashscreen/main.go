// Command splashscreen follows the same composition-only shape as
// examples/welcomescreen and examples/loginflow: widgets.BigText for the
// logo and widgets.Header for a tagline, dismissed by any keypress. It
// introduces no new pattern beyond what those two examples already
// demonstrate.
package main

import (
	"fmt"
	"os"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/theme"
	"github.com/ows4444/tui/widgets"
)

type model struct {
	t theme.Theme
}

func initialModel() model {
	return model{t: theme.DarkTheme()}
}

func (m model) Init() tui.Cmd { return nil }

func (m model) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	if _, ok := msg.(tui.Key); ok {
		return m, tui.Quit()
	}
	return m, nil
}

var tagline = ansi.NewStyle().Faint()

// screen stacks the logo, the header and the hint with one blank row between
// them, as a layout.Node.
func (m model) screen() layout.Node {
	return layout.Column(1,
		layout.FlexChild{Node: layout.Block(widgets.BigText("TUI", widgets.FontBlock, m.t))},
		layout.FlexChild{Node: layout.Block(widgets.Header("Loading your workspace…", m.t))},
		layout.FlexChild{Node: layout.Block(tagline.Render("press any key to continue"))},
	)
}

func (m model) View() string { return layout.Draw(m.screen(), layout.Unconstrained()) }

func main() {
	if _, err := tui.NewProgram(initialModel()).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
