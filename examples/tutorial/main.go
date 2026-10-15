// Command tutorial is the program built step by step in docs/tutorial.md: a
// task list with a text field, a cursor and a done mark.
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/textinput"
)

type task struct {
	title string
	done  bool
}

type model struct {
	input  textinput.Model
	tasks  []task
	cursor int
	width  int
}

func newModel() model {
	in := textinput.New()
	in.Prompt = "> "
	in.Placeholder = "What needs doing?"
	in.Width = 30
	in.Focus() // the blink Cmd comes back from Init
	return model{input: in}
}

func (m model) Init() tui.Cmd { return m.input.Focus() }

func (m model) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	switch msg := msg.(type) {
	case tui.ResizeMsg:
		m.width = msg.Width
		return m, nil
	case tui.Key:
		switch msg.Type {
		case tui.KeyCtrlC:
			return m, tui.Quit()
		case tui.KeyEnter:
			if title := strings.TrimSpace(m.input.Value()); title != "" {
				m.tasks = append(m.tasks, task{title: title})
				m.input.Reset()
			}
			return m, nil
		case tui.KeyUp:
			m.cursor = max(m.cursor-1, 0)
			return m, nil
		case tui.KeyDown:
			m.cursor = max(min(m.cursor+1, len(m.tasks)-1), 0)
			return m, nil
		case tui.KeyTab:
			if m.cursor < len(m.tasks) {
				m.tasks[m.cursor].done = !m.tasks[m.cursor].done
			}
			return m, nil
		}
	}
	var cmd tui.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

var (
	titleStyle = ansi.NewStyle().Bold().Foreground(ansi.BrightCyan)
	doneStyle  = ansi.NewStyle().Faint().Strikethrough()
	helpStyle  = ansi.NewStyle().Faint()
)

func (m model) View() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Tasks"))
	b.WriteString("\n\n")
	b.WriteString(m.input.View())
	b.WriteString("\n\n")
	if len(m.tasks) == 0 {
		b.WriteString(helpStyle.Render(m.fit("Nothing yet. Type a task and press enter.")))
		b.WriteString("\n")
	}
	for i, t := range m.tasks {
		cursor, box, title := "  ", "[ ]", t.title
		if i == m.cursor {
			cursor = "> "
		}
		if t.done {
			box, title = "[x]", doneStyle.Render(t.title)
		}
		fmt.Fprintf(&b, "%s%s %s\n", cursor, box, title)
	}
	b.WriteString("\n")
	b.WriteString(helpStyle.Render(m.fit("enter add   up/down move   tab done   ctrl+c quit")))
	return b.String()
}

// fit wraps s to the terminal's width once the size is known.
func (m model) fit(s string) string {
	if m.width > 0 {
		return ansi.Wrap(s, m.width)
	}
	return s
}

func main() {
	if _, err := tui.NewProgram(newModel()).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
