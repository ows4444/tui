// Command docsreader is a Markdown reader with a table of contents: the
// document scrolls in markdownview, the headings are listed beside it, and Tab
// follows the links (an in-page link scrolls, Backspace goes back). It shows
// how markdownview.Headings and Links feed a side panel, and how a LinkMsg
// leaves the program to open.
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/markdownview"
)

const doc = `# tui

A terminal UI library for Go. Start with the [quick start](#quick-start), or read
about the [architecture](#architecture). The source is on [GitHub](https://github.com/ows4444/tui).

## Quick start

Write a Model with Init, Update and View, hand it to a Program, and call Run.

- [x] a model
- [x] a program
- [ ] your idea

## Architecture

A Program owns the terminal and one event loop. Messages go in, a View comes out,
and the renderer writes only what changed. See [quick start](#quick-start) again.

## Testing

tuitest runs a model headless: send keys, read the screen, compare a golden file.
`

type model struct {
	reader markdownview.Model
	opened string
	w, h   int // terminal size from the last ResizeMsg; 80x24 until known
}

// readerSize is the window the reader gets: the terminal less the contents
// panel and the status row.
func readerSize(w, h int) (int, int) { return max(w-24, 20), max(h-3, 4) }

func initialModel() model {
	rw, rh := readerSize(80, 24)
	return model{reader: markdownview.New(doc, rw, rh), w: 80, h: 24}
}

func (m model) Init() tui.Cmd { return nil }

func (m model) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	switch msg := msg.(type) {
	case tui.ResizeMsg:
		m.w, m.h = msg.Width, msg.Height
		m.reader.SetSize(readerSize(msg.Width, msg.Height))
		return m, nil
	case markdownview.LinkMsg:
		m.opened = msg.URL // a real program would open it with the system browser
		return m, nil
	case tui.Key:
		if msg.Type == tui.KeyCtrlC || msg.Type == tui.KeyEsc || (msg.Type == tui.KeyRunes && msg.Text == "q") {
			return m, tui.Quit()
		}
	}
	var cmd tui.Cmd
	m.reader, cmd = m.reader.Update(msg)
	return m, cmd
}

// contents lists the headings, indented by level; the nearest above the top
// of the window is marked.
func (m model) contents() string {
	var b strings.Builder
	b.WriteString("Contents\n")
	for _, h := range m.reader.Headings() {
		mark := "  "
		if h.Line <= m.reader.YOffset() {
			mark = "> "
		}
		b.WriteString(mark + strings.Repeat("  ", h.Level-1) + h.Text + "\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

func (m model) screen() layout.Node {
	status := "tab: next link   enter: follow   ]/[: heading   backspace: back   q: quit"
	if m.opened != "" {
		status = "would open " + m.opened
	}
	return layout.Column(1,
		layout.Fill(layout.Row(2,
			layout.FlexChild{Node: layout.Block(m.contents()), Basis: 22},
			layout.Fill(m.reader.LayoutNode()),
		)),
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
