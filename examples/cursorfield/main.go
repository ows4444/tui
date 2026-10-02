// Command cursorfield shows a text input whose cursor is the terminal's real
// cursor: the model implements tui.CursorPlacer by forwarding
// textinput.Model.CursorCell, so an IME or a screen magnifier can follow the
// field. Type text, Enter or Esc quits.
package main

import (
	"fmt"
	"os"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/textinput"
)

// title is the row above the field; the field starts on the next row.
const title = "Name (Enter quits):"

type model struct{ in textinput.Model }

func initialModel() model {
	in := textinput.New()
	in.Prompt = "> "
	in.Focus()
	return model{in: in}
}

func (m model) Init() tui.Cmd { return nil }

func (m model) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	if k, ok := msg.(tui.Key); ok {
		switch k.Type {
		case tui.KeyCtrlC, tui.KeyEsc, tui.KeyEnter:
			return m, tui.Quit()
		}
	}
	var cmd tui.Cmd
	m.in, cmd = m.in.Update(msg)
	return m, cmd
}

func (m model) View() string { return title + "\n" + m.in.View() }

// CursorPos implements tui.CursorPlacer: the field's cell, moved down past
// the title row.
func (m model) CursorPos() (x, y int, ok bool) {
	x, y, ok = m.in.CursorCell()
	return x, y + 1, ok
}

func main() {
	if _, err := tui.NewProgram(initialModel(), tui.WithAltScreen(false)).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
