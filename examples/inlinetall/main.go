// Command inlinetall is an inline-mode (no alternate screen) transcript that
// grows past the terminal height. Rows that no longer fit are committed to
// scrollback once; only the visible tail stays live. Press Enter to append a
// line, Esc or Ctrl+C to quit.
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
)

type model struct {
	lines []string
	width int // terminal width from the last ResizeMsg; 0 until known
}

func initialModel() model { return model{lines: []string{"transcript (Enter adds a line, Esc quits)"}} }

func (m model) Init() tui.Cmd { return nil }

func (m model) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	if rs, ok := msg.(tui.ResizeMsg); ok {
		m.width = rs.Width
		return m, nil
	}
	if k, ok := msg.(tui.Key); ok {
		switch k.Type {
		case tui.KeyCtrlC, tui.KeyEsc:
			return m, tui.Quit()
		case tui.KeyEnter:
			m.lines = append(append([]string(nil), m.lines...), fmt.Sprintf("line %d", len(m.lines)))
		}
	}
	return m, nil
}

// View is the transcript, word-wrapped to the terminal width so a long line
// takes two rows instead of overflowing.
func (m model) View() string {
	text := strings.Join(m.lines, "\n")
	if m.width > 0 {
		text = ansi.Wrap(text, m.width)
	}
	return text
}

func main() {
	if _, err := tui.NewProgram(initialModel(), tui.WithAltScreen(false)).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
