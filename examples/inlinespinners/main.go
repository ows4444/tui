// Command inlinespinners is an inline-mode (no alternate screen) list of
// tasks: finished tasks show a check, the running one a spinner, the rest
// wait. The whole list is the live region.
package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/motion"
	"github.com/ows4444/tui/spinner"
)

var tasks = []string{"resolve", "fetch", "build", "verify"}

type doneMsg struct{ i int }

type model struct {
	sp   spinner.Model
	done int
	// start is the spinner's first tick. Start has to run on the model the
	// Program keeps, and Init has a value receiver and gets a copy, so it
	// runs where the model is built and Init only hands the Cmd over.
	start tui.Cmd
}

func initialModel() model {
	m := model{sp: spinner.New()}
	m.start = m.sp.Start()
	return m
}

func (m model) Init() tui.Cmd {
	cmds := []tui.Cmd{m.start}
	for i := range tasks {
		i := i
		cmds = append(cmds, tui.FromCtx(motion.After(time.Duration(i+1)*600*time.Millisecond,
			func(time.Time) tui.Msg { return doneMsg{i} })))
	}
	return tui.Batch(cmds...)
}

func (m model) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	switch msg := msg.(type) {
	case tui.Key:
		if msg.Type == tui.KeyCtrlC || msg.Type == tui.KeyEsc {
			return m, tui.Quit()
		}
	case doneMsg:
		if msg.i+1 > m.done {
			m.done = msg.i + 1
		}
		if m.done >= len(tasks) {
			m.sp.Stop()
			return m, tui.Quit()
		}
		return m, nil
	default:
		var cmd tui.Cmd
		m.sp, cmd = m.sp.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m model) View() string {
	rows := make([]string, len(tasks))
	for i, name := range tasks {
		switch {
		case i < m.done:
			rows[i] = "[x] " + name
		case i == m.done:
			rows[i] = m.sp.View() + " " + name
		default:
			rows[i] = "[ ] " + name
		}
	}
	return strings.Join(rows, "\n")
}

func main() {
	if _, err := tui.NewProgram(initialModel(), tui.WithAltScreen(false)).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
