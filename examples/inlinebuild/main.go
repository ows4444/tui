// Command inlinebuild is an inline-mode (no alternate screen) build log.
// Finished lines are committed to scrollback with tui.Println; the live
// region is a fixed-height tail of the most recent output plus a status row.
package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/motion"
)

const tailRows = 4

var lines = []string{
	"go: downloading modules", "compile internal/ansi", "compile layout",
	"compile widgets", "compile theme", "link tui", "test ./... ok", "package done",
}

type lineMsg struct{ i int }

type model struct {
	tail []string
	done bool
}

func (m model) Init() tui.Cmd {
	cmds := make([]tui.Cmd, len(lines))
	for i := range lines {
		i := i
		cmds[i] = tui.FromCtx(motion.After(time.Duration(i+1)*200*time.Millisecond,
			func(time.Time) tui.Msg { return lineMsg{i} }))
	}
	return tui.Batch(cmds...)
}

func (m model) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	switch msg := msg.(type) {
	case tui.Key:
		if msg.Type == tui.KeyCtrlC || msg.Type == tui.KeyEsc {
			return m, tui.Quit()
		}
	case lineMsg:
		m.tail = append(m.tail, lines[msg.i])
		if len(m.tail) > tailRows {
			m.tail = m.tail[len(m.tail)-tailRows:]
		}
		if msg.i == len(lines)-1 {
			m.done = true
			return m, tui.Sequence(tui.Println("ok "+lines[msg.i]), tui.Quit())
		}
		return m, tui.Println("ok " + lines[msg.i])
	}
	return m, nil
}

func (m model) View() string {
	status := "building..."
	if m.done {
		status = "build finished"
	}
	rows := append([]string{status}, m.tail...)
	return strings.Join(rows, "\n")
}

func main() {
	if _, err := tui.NewProgram(model{}, tui.WithAltScreen(false)).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
