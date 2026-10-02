package main

import (
	"fmt"
	"os"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
)

type model struct {
	count         int
	width, height int
}

func (m model) Init() tui.Cmd { return nil }

func (m model) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	switch msg := msg.(type) {
	case tui.ResizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case tui.Key:
		switch msg.Type {
		case tui.KeyCtrlC:
			return m, tui.Quit()
		case tui.KeyUp, tui.KeyRight:
			m.count++
		case tui.KeyDown, tui.KeyLeft:
			m.count--
		case tui.KeyRunes:
			if msg.Text == "q" {
				return m, tui.Quit()
			}
		}
	}
	return m, nil
}

var (
	titleStyle = ansi.NewStyle().Bold().Foreground(ansi.BrightCyan)
	countStyle = ansi.NewStyle().Bold().Foreground(ansi.BrightGreen)
	helpStyle  = ansi.NewStyle().Faint()
)

func (m model) View() string {
	help := fmt.Sprintf("(%dx%d)  ↑/→ increment  ↓/← decrement  q/ctrl+c quit", m.width, m.height)
	if m.width > 0 {
		help = ansi.Wrap(help, m.width)
	}
	return fmt.Sprintf(
		"%s\n\n  Count: %s\n\n%s",
		titleStyle.Render("From-scratch TUI — Counter"),
		countStyle.Render(fmt.Sprintf("%d", m.count)),
		helpStyle.Render(help),
	)
}

func main() {
	if _, err := tui.NewProgram(model{}).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
