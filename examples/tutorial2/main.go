// Command tutorial2 is the program built in docs/tutorial-2.md: the task list
// of the first tutorial with a list that narrows as you type and a column of
// counts, using picker's Filter.
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/picker"
	"github.com/ows4444/tui/textinput"
)

type task struct {
	title string
	done  bool
}

type model struct {
	input   textinput.Model
	list    picker.Model
	tasks   []task
	inList  bool // focus: the list when true, the text field otherwise
	width   int
	pending int // tasks not done, for the status line
}

func newModel() model {
	in := textinput.New()
	in.Prompt = "> "
	in.Placeholder = "What needs doing?"
	in.Width = 30
	in.Focus() // the blink Cmd comes back from Init
	m := model{input: in}
	m.sync("")
	return m
}

func (m model) Init() tui.Cmd { return m.input.Focus() }

// sync rebuilds the list from the tasks, keeping the query and the cursor.
func (m *model) sync(query string) {
	cursor := m.list.Cursor()
	labels := make([]string, len(m.tasks))
	m.pending = 0
	for i, t := range m.tasks {
		box := "[ ]"
		if t.done {
			box = "[x]"
		} else {
			m.pending++
		}
		labels[i] = box + " " + t.title
	}
	m.list = picker.NewStrings(labels...)
	m.list.Filter = true
	m.list.SetQuery(query)
	m.list.SetCursor(cursor)
}

func (m model) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	switch msg := msg.(type) {
	case tui.ResizeMsg:
		m.width = msg.Width
		return m, nil
	case picker.SelectedMsg:
		// Enter on a row of the list: the task it stands for is done, or not.
		m.tasks[msg.Index].done = !m.tasks[msg.Index].done
		m.sync(m.list.Query())
		return m, nil
	case tui.Key:
		switch msg.Type {
		case tui.KeyCtrlC:
			return m, tui.Quit()
		case tui.KeyTab:
			m.inList = !m.inList
			if m.inList {
				m.input.Blur()
				return m, nil
			}
			return m, m.input.Focus()
		case tui.KeyEnter:
			if !m.inList {
				if title := strings.TrimSpace(m.input.Value()); title != "" {
					m.tasks = append(m.tasks, task{title: title})
					m.input.Reset()
					m.sync(m.list.Query())
				}
				return m, nil
			}
		}
	}
	var cmd tui.Cmd
	if m.inList {
		m.list, cmd = m.list.Update(msg)
	} else {
		m.input, cmd = m.input.Update(msg)
	}
	return m, cmd
}

var (
	titleStyle = ansi.NewStyle().Bold().Foreground(ansi.BrightCyan)
	helpStyle  = ansi.NewStyle().Faint()
)

func (m model) View() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Tasks") + "\n\n")
	b.WriteString(m.input.View() + "\n\n")
	if len(m.tasks) == 0 {
		b.WriteString(helpStyle.Render("Nothing yet. Type a task and press enter.") + "\n")
	} else {
		b.WriteString(m.list.View() + "\n")
	}
	fmt.Fprintf(&b, "\n%s\n", helpStyle.Render(fmt.Sprintf("%d of %d left", m.pending, len(m.tasks))))
	help := "enter add   tab list   type to search   enter toggle   ctrl+c quit"
	if m.width > 0 {
		help = ansi.Wrap(help, m.width)
	}
	b.WriteString(helpStyle.Render(help))
	return b.String()
}

func main() {
	if _, err := tui.NewProgram(newModel()).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
