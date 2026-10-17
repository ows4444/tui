// Command logbrowser filters a log as you type. The filter is a textinput with
// History (Up and Down recall earlier filters, Ctrl+R searches them) and a
// suggestion from History, and the matching lines scroll in a viewport. Enter
// keeps the filter in History.
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/textinput"
	"github.com/ows4444/tui/viewport"
)

var levels = []string{"INFO", "WARN", "ERROR", "INFO", "INFO"}
var sources = []string{"api", "worker", "gateway"}

// logLines is 200 generated lines; each is computed from its index alone.
func logLines() []string {
	out := make([]string, 200)
	for i := range out {
		out[i] = fmt.Sprintf("%02d:%02d:%02d %-5s %-7s request %d handled", i/3600%24, i/60%60, i%60, levels[i%len(levels)], sources[i%len(sources)], i)
	}
	return out
}

type model struct {
	filter  textinput.Model
	view    viewport.Model
	lines   []string
	matches int
	w, h    int // terminal size from the last ResizeMsg; 80x24 until known
}

func initialModel() model {
	f := textinput.New()
	f.Prompt = "filter> "
	f.History = []string{"ERROR", "worker", "WARN gateway"}
	f.SuggestFromHistory = true
	f.Focus()
	m := model{filter: f, lines: logLines(), w: 80, h: 24}
	return m.fit()
}

// fit sizes the viewport to the terminal and refills it from the filter.
func (m model) fit() model {
	m.view.Width, m.view.Height = max(m.w, 1), max(m.h-3, 1)
	shown := []string{}
	for _, l := range m.lines {
		if strings.Contains(strings.ToLower(l), strings.ToLower(m.filter.Value())) {
			shown = append(shown, l)
		}
	}
	m.matches = len(shown)
	m.view.SetContent(strings.Join(shown, "\n"))
	return m
}

func (m model) Init() tui.Cmd { return nil }

func (m model) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	switch msg := msg.(type) {
	case tui.ResizeMsg:
		m.w, m.h = msg.Width, msg.Height
		return m.fit(), nil
	case tui.Key:
		switch {
		case msg.Type == tui.KeyCtrlC || msg.Type == tui.KeyEsc:
			return m, tui.Quit()
		case msg.Type == tui.KeyEnter:
			if v := m.filter.Value(); v != "" {
				m.filter.History = append(m.filter.History, v)
			}
			return m, nil
		case msg.Type == tui.KeyPgUp || msg.Type == tui.KeyPgDown:
			var cmd tui.Cmd
			m.view, cmd = m.view.Update(msg)
			return m, cmd
		}
	}
	var cmd tui.Cmd
	m.filter, cmd = m.filter.Update(msg)
	return m.fit(), cmd
}

func (m model) screen() layout.Node {
	status := fmt.Sprintf("%d of %d lines   up/down: history   ctrl+r: search   pgup/pgdn: scroll   esc: quit", m.matches, len(m.lines))
	return layout.Column(0,
		layout.FlexChild{Node: m.filter.LayoutNode()},
		layout.FlexChild{Node: layout.Block(strings.Repeat("─", max(m.w, 1)))},
		layout.Fill(m.view.LayoutNode()),
		layout.FlexChild{Node: layout.Text(status, layout.WithEllipsis("…"))},
	)
}

func (m model) View() string {
	return layout.Draw(m.screen(), layout.Tight(layout.Size{W: m.w, H: m.h}))
}

func main() {
	if _, err := tui.NewProgram(initialModel(), tui.WithAltScreen(true)).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
